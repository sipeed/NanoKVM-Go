// Feed browser JPEGs to the NanoKVM Go+ UVC 1.0 output.
#define _DEFAULT_SOURCE
#include <endian.h>
#include <errno.h>
#include <fcntl.h>
#include <linux/usb/g_uvc.h>
#include <linux/usb/video.h>
#include <linux/videodev2.h>
#include <poll.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <sys/timerfd.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/un.h>
#include <time.h>
#include <unistd.h>

enum { FRAME_INDEX = 1, MAX_FRAME = 131072, PACKET = 512,
       INTERVAL_30 = 333333, CONTROL_SIZE = 26, BUFFER_COUNT = 3 };
static const enum v4l2_buf_type output = V4L2_BUF_TYPE_VIDEO_OUTPUT;

struct slot {
    void *data;
    size_t length;
    int queued;
};
struct camera {
    int fd, timer, control_if, stream_if, streaming, live_fd;
    unsigned count;
    uint8_t error;
    struct slot slots[BUFFER_COUNT];
    uint8_t latest[MAX_FRAME], idle[MAX_FRAME];
    size_t latest_len, idle_len;
    struct timespec received;
};

static void die(const char *message)
{
    perror(message);
    exit(1);
}

static int xioctl(int fd, unsigned long request, void *arg)
{
    int rc;
    do { rc = ioctl(fd, request, arg); } while (rc < 0 && errno == EINTR);
    return rc;
}

static int valid_jpeg(const uint8_t *data, size_t size)
{
    return size >= 4 && size <= MAX_FRAME && data[0] == 0xff && data[1] == 0xd8 &&
           data[size - 2] == 0xff && data[size - 1] == 0xd9;
}

static void load_idle_frame(struct camera *cam, const char *path)
{
    FILE *file = fopen(path, "rb");
    if (!file) die("open camera-off JPEG");
    cam->idle_len = fread(cam->idle, 1, sizeof(cam->idle), file);
    int extra = fgetc(file);
    if (ferror(file)) die("read camera-off JPEG");
    fclose(file);
    if (extra != EOF || !valid_jpeg(cam->idle, cam->idle_len)) {
        fprintf(stderr, "Invalid camera-off JPEG\n");
        exit(1);
    }
}

static void put32(uint8_t *dst, uint32_t value)
{
    value = htole32(value);
    memcpy(dst, &value, sizeof(value));
}

static void profile(uint8_t *data)
{
    data[0] = 1; // Fixed frame interval.
    data[2] = 1; // MJPEG format.
    data[3] = FRAME_INDEX;
    put32(data + 4, INTERVAL_30);
    put32(data + 18, MAX_FRAME);
    put32(data + 22, PACKET);
}

static struct uvc_request_data setup(struct camera *cam, const struct usb_ctrlrequest *req)
{
    struct uvc_request_data r = { .length = -EL2HLT };
    int index = le16toh(req->wIndex);
    int iface = index & 0xff, entity = index >> 8;
    int selector = le16toh(req->wValue) >> 8, length = le16toh(req->wLength);
    if (req->bRequestType != 0xa1 && req->bRequestType != 0x21)
        return r;
    if (iface == cam->stream_if && entity == 0 &&
        (selector == UVC_VS_PROBE_CONTROL || selector == UVC_VS_COMMIT_CONTROL)) {
        if (req->bRequestType == 0x21) {
            if (req->bRequest == UVC_SET_CUR && length == CONTROL_SIZE)
                r.length = length;
        } else {
            switch (req->bRequest) {
            case UVC_GET_CUR: case UVC_GET_MIN: case UVC_GET_MAX: case UVC_GET_DEF:
                profile(r.data);
                r.length = CONTROL_SIZE;
                break;
            case UVC_GET_RES: r.length = CONTROL_SIZE; break;
            case UVC_GET_LEN: r.data[0] = CONTROL_SIZE; r.length = 2; break;
            case UVC_GET_INFO: r.data[0] = 3; r.length = 1; break;
            }
        }
    } else if (iface == cam->control_if && req->bRequestType == 0xa1) {
        // The pinned kernel advertises AE mode and brightness. The virtual camera has
        // fixed exposure and brightness and reports both controls as read-only.
        if (!entity && selector == UVC_VC_REQUEST_ERROR_CODE_CONTROL && req->bRequest == UVC_GET_CUR) {
            r.data[0] = cam->error;
            r.length = 1;
        } else if ((entity == 1 && selector == UVC_CT_AE_MODE_CONTROL) ||
                   (entity == 2 && selector == UVC_PU_BRIGHTNESS_CONTROL)) {
            int width = entity == 1 ? 1 : 2;
            switch (req->bRequest) {
            case UVC_GET_INFO: r.data[0] = 1; r.length = 1; break;
            case UVC_GET_LEN: r.data[0] = width; r.length = 2; break;
            case UVC_GET_CUR: case UVC_GET_DEF: case UVC_GET_MIN: case UVC_GET_MAX: case UVC_GET_RES:
                r.data[0] = entity == 1 ? 2 : (req->bRequest == UVC_GET_RES);
                r.length = width;
                break;
            }
        }
    }
    cam->error = r.length < 0 ? 0x07 : 0;
    if (r.length > length)
        r.length = length;
    return r;
}

// Keep the compressed image only in RAM. A stale producer cannot leave a
// private image frozen on the host if the browser or server disappears.
static size_t live_frame(struct camera *cam, void *dest)
{
    struct timespec now;
    clock_gettime(CLOCK_MONOTONIC, &now);
    int64_t age = (int64_t)(now.tv_sec - cam->received.tv_sec) * 1000000000 +
                  now.tv_nsec - cam->received.tv_nsec;
    if (age >= 1000000000) cam->latest_len = 0;
    size_t n = cam->latest_len ? cam->latest_len : cam->idle_len;
    memcpy(dest, cam->latest_len ? cam->latest : cam->idle, n);
    return n;
}

static const char live_socket[] = "/run/nanokvm-webcam/live.sock";
static const char stream_flag[] = "/run/nanokvm-webcam/streaming";

static void start_live_input(struct camera *cam)
{
    struct sockaddr_un addr = { .sun_family = AF_UNIX };
    memcpy(addr.sun_path, live_socket, sizeof(live_socket));
    cam->live_fd = socket(AF_UNIX, SOCK_DGRAM | SOCK_NONBLOCK | SOCK_CLOEXEC, 0);
    if (cam->live_fd < 0) die("create webcam input");
    unlink(stream_flag);
    unlink(live_socket);
    if (bind(cam->live_fd, (struct sockaddr *)&addr, sizeof(addr)) < 0) die("bind webcam input");
    if (chmod(live_socket, 0600) < 0) die("restrict webcam input");
}

static void receive_live_frame(struct camera *cam)
{
    uint8_t index = 0;
    struct iovec parts[] = {{ &index, 1 }, { cam->latest, sizeof(cam->latest) }};
    struct msghdr msg = { .msg_iov = parts, .msg_iovlen = 2 };
    ssize_t n = recvmsg(cam->live_fd, &msg, MSG_TRUNC);
    if (n < 0) {
        if (errno == EAGAIN || errno == EINTR) return;
        die("receive webcam frame");
    }
    cam->latest_len = 0;
    // The server sends a single zero byte to clear the image on disconnect.
    if (!cam->streaming || index != FRAME_INDEX || n < 1 ||
        !valid_jpeg(cam->latest, (size_t)n - 1))
        return;
    cam->latest_len = (size_t)n - 1;
    clock_gettime(CLOCK_MONOTONIC, &cam->received);
}

static void send_frame(struct camera *cam)
{
    for (unsigned i = 0; i < cam->count; i++) {
        struct slot *slot = &cam->slots[i];
        if (slot->queued)
            continue;
        size_t n = live_frame(cam, slot->data);
        struct v4l2_buffer b = { .type = output, .memory = V4L2_MEMORY_MMAP,
                                 .index = i, .bytesused = n };
        if (xioctl(cam->fd, VIDIOC_QBUF, &b) < 0) {
            if (errno == ENODEV && cam->streaming)
                return;
            die("queue JPEG");
        }
        slot->queued = 1;
        return;
    }
}

static void start_stream(struct camera *cam)
{
    if (cam->streaming)
        return;
    struct v4l2_format fmt = { .type = output, .fmt.pix = {
        .width = 1024, .height = 600, .pixelformat = V4L2_PIX_FMT_MJPEG,
        .field = V4L2_FIELD_NONE, .sizeimage = MAX_FRAME } };
    if (xioctl(cam->fd, VIDIOC_S_FMT, &fmt) < 0)
        die("set MJPEG format");
    struct v4l2_requestbuffers buffers = {
        .type = output, .memory = V4L2_MEMORY_MMAP, .count = BUFFER_COUNT };
    if (xioctl(cam->fd, VIDIOC_REQBUFS, &buffers) < 0 || !buffers.count || buffers.count > BUFFER_COUNT)
        die("allocate output buffers");
    cam->count = buffers.count;
    for (unsigned i = 0; i < cam->count; i++) {
        struct v4l2_buffer b = { .type = output, .memory = V4L2_MEMORY_MMAP, .index = i };
        if (xioctl(cam->fd, VIDIOC_QUERYBUF, &b) < 0 || b.length < MAX_FRAME)
            die("query output buffer");
        cam->slots[i].length = b.length;
        cam->slots[i].queued = 0;
        cam->slots[i].data = mmap(NULL, b.length, PROT_READ | PROT_WRITE, MAP_SHARED, cam->fd, b.m.offset);
        if (cam->slots[i].data == MAP_FAILED)
            die("map output buffer");
    }
    send_frame(cam);
    enum v4l2_buf_type type = output;
    if (xioctl(cam->fd, VIDIOC_STREAMON, &type) < 0)
        die("start isochronous stream");
    long tick_ns = INTERVAL_30 * 100L;
    struct itimerspec tick = { .it_value.tv_nsec = tick_ns, .it_interval.tv_nsec = tick_ns };
    if (timerfd_settime(cam->timer, 0, &tick, NULL) < 0)
        die("start frame clock");
    cam->streaming = 1;
    if (symlink("600p30", stream_flag) < 0) die("publish webcam demand");
    fprintf(stderr, "Streaming %ux%u MJPEG at 30 fps\n", fmt.fmt.pix.width, fmt.fmt.pix.height);
}

static void stop_stream(struct camera *cam)
{
    if (!cam->streaming)
        return;
    unlink(stream_flag);
    cam->latest_len = 0;
    struct itimerspec off = {0};
    if (timerfd_settime(cam->timer, 0, &off, NULL) < 0)
        die("stop frame clock");
    enum v4l2_buf_type type = output;
    if (xioctl(cam->fd, VIDIOC_STREAMOFF, &type) < 0 && errno != ENODEV)
        die("stop stream");
    for (unsigned i = 0; i < cam->count; i++)
        munmap(cam->slots[i].data, cam->slots[i].length);
    struct v4l2_requestbuffers buffers = { .type = output, .memory = V4L2_MEMORY_MMAP };
    if (xioctl(cam->fd, VIDIOC_REQBUFS, &buffers) < 0)
        die("release buffers");
    cam->streaming = 0;
}

static void handle_event(struct camera *cam)
{
    struct v4l2_event v;
    if (xioctl(cam->fd, VIDIOC_DQEVENT, &v) < 0) {
        if (errno == EAGAIN) return;
        die("dequeue UVC event");
    }
    struct uvc_event e;
    memcpy(&e, v.u.data, sizeof(e));
    switch (v.type) {
    case UVC_EVENT_SETUP: {
        struct uvc_request_data r = setup(cam, &e.req);
        if (xioctl(cam->fd, UVCIOC_SEND_RESPONSE, &r) < 0 && errno != ENODEV && errno != ESHUTDOWN)
            die("reply to UVC request");
        break;
    }
    case UVC_EVENT_DATA:
        // Probe/Commit can only select the single profile returned by setup().
        break;
    case UVC_EVENT_STREAMON: start_stream(cam); break;
    case UVC_EVENT_DISCONNECT:
        // Invalidate any browser publisher when the physical USB host disconnects.
        if (utimensat(AT_FDCWD, live_socket, NULL, 0) < 0)
            die("invalidate webcam session");
        stop_stream(cam);
        break;
    case UVC_EVENT_STREAMOFF: stop_stream(cam); break;
    }
}

static void frame_tick(struct camera *cam)
{
    uint64_t ticks;
    if (read(cam->timer, &ticks, sizeof(ticks)) != sizeof(ticks)) {
        if (errno == EAGAIN) return;
        die("frame clock");
    }
    struct v4l2_buffer b = { .type = output, .memory = V4L2_MEMORY_MMAP };
    while (xioctl(cam->fd, VIDIOC_DQBUF, &b) >= 0) {
        if (b.index >= cam->count) { fprintf(stderr, "Invalid buffer index\n"); exit(1); }
        cam->slots[b.index].queued = 0;
    }
    if (errno == EINVAL || errno == ENODEV) return; // Kernel stop event is pending.
    if (errno != EAGAIN) die("dequeue output buffer");
    send_frame(cam);
}

int main(int argc, char **argv)
{
    if (argc != 5) {
        fprintf(stderr, "Usage: %s DEVICE IDLE_JPEG CONTROL_IF STREAM_IF\n", argv[0]);
        return 2;
    }
    struct camera cam = {0};
    load_idle_frame(&cam, argv[2]);
    cam.control_if = atoi(argv[3]);
    cam.stream_if = atoi(argv[4]);
    cam.fd = open(argv[1], O_RDWR | O_NONBLOCK | O_CLOEXEC);
    if (cam.fd < 0) die("open UVC output");
    struct v4l2_capability cap = {0};
    if (xioctl(cam.fd, VIDIOC_QUERYCAP, &cap) < 0 || strcmp((char *)cap.driver, "g_uvc") ||
        !(cap.device_caps & V4L2_CAP_VIDEO_OUTPUT) || !(cap.device_caps & V4L2_CAP_STREAMING))
        die("check UVC output");
    for (unsigned type = UVC_EVENT_FIRST; type <= UVC_EVENT_LAST; type++) {
        struct v4l2_event_subscription sub = { .type = type };
        if (xioctl(cam.fd, VIDIOC_SUBSCRIBE_EVENT, &sub) < 0) die("subscribe UVC event");
    }
    cam.timer = timerfd_create(CLOCK_MONOTONIC, TFD_CLOEXEC | TFD_NONBLOCK);
    if (cam.timer < 0) die("create frame clock");
    start_live_input(&cam);
    struct pollfd fds[] = {{ .fd = cam.fd, .events = POLLPRI }, { .fd = cam.timer, .events = POLLIN },
                           { .fd = cam.live_fd, .events = POLLIN }};
    fprintf(stderr, "Ready on %s, control %d, streaming %d\n", argv[1], cam.control_if, cam.stream_if);
    for (;;) {
        if (poll(fds, 3, -1) < 0) {
            if (errno == EINTR) continue;
            die("poll camera");
        }
        if (fds[0].revents & (POLLHUP | POLLNVAL)) return 1;
        if (fds[0].revents & POLLPRI) handle_event(&cam);
        if (fds[2].revents & POLLIN) receive_live_frame(&cam);
        if (cam.streaming && (fds[1].revents & POLLIN)) frame_tick(&cam);
    }
}
