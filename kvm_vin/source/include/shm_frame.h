#ifndef SHM_FRAME_H
#define SHM_FRAME_H

#include <errno.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
#include <sys/file.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

#ifdef __cplusplus
extern "C" {
#endif

#define SHM_FRAME_MAGIC         0x4B564D46U  /* "KVMF" */
#define SHM_FRAME_VERSION       2
#define SHM_H264_NAME           "/kvm_h264"
/* The shipped kvm_vin producer and libkvm reader run as the same root owner. */
#define SHM_FRAME_MODE          (S_IRUSR | S_IWUSR)
#define SHM_FRAME_CREATE_FLAGS  (O_CREAT | O_EXCL | O_RDWR)

#define SHM_H264_MAX_FRAME_SIZE     (2U * 1024U * 1024U)   /* 2 MB */

#define SHM_FRAME_WRITE_OK      0
#define SHM_FRAME_WRITE_ERROR  -1
#define SHM_FRAME_WRITE_BUSY   -2

typedef struct {
    uint64_t frame_id;      /* monotonically increasing latest-frame id */
    uint32_t data_len;
    uint32_t width;
    uint32_t height;
    uint32_t format;        /* V4L2_PIX_FMT_H264 / V4L2_PIX_FMT_HEVC */
    uint8_t  key_frame;
    uint8_t  _pad1[7];
    uint64_t pts_us;        /* monotonic timestamp in microseconds */
} shm_frame_slot_t;

typedef struct {
    uint32_t magic;
    uint32_t version;
    uint32_t max_frame_size;
    uint32_t src_width;     /* source capture resolution (informational) */
    uint32_t src_height;
    uint32_t src_fps;
    uint32_t _pad;
    uint64_t reader_heartbeat_us;
    shm_frame_slot_t frame;
    uint8_t data[];         /* flexible array, actual size = max_frame_size */
} shm_frame_ring_t;

typedef struct {
    int fd;
    size_t size;
    ino_t ino;
    shm_frame_ring_t *ring;
} shm_frame_handle_t;

static inline size_t shm_frame_size(uint32_t max_frame_size)
{
    size_t raw = offsetof(shm_frame_ring_t, data) + max_frame_size;
    return (raw + 7U) & ~(size_t)7U;  /* align to 8 bytes */
}

static inline shm_frame_ring_t *shm_frame_ring(shm_frame_handle_t *handle)
{
    return handle ? handle->ring : NULL;
}

static inline const shm_frame_ring_t *shm_frame_ring_rd(const shm_frame_handle_t *handle)
{
    return handle ? handle->ring : NULL;
}

static inline shm_frame_handle_t *shm_frame_alloc_handle(int fd, size_t size, ino_t ino, void *mapping)
{
    shm_frame_handle_t *handle = (shm_frame_handle_t *)calloc(1, sizeof(shm_frame_handle_t));
    if (!handle) return NULL;

    handle->fd = fd;
    handle->size = size;
    handle->ino = ino;
    handle->ring = (shm_frame_ring_t *)mapping;
    return handle;
}

static inline shm_frame_handle_t *shm_frame_create(const char *name, uint32_t max_frame_size)
{
    shm_unlink(name);
    /* Fail closed if another principal recreates the name after unlink. */
    int fd = shm_open(name, SHM_FRAME_CREATE_FLAGS, SHM_FRAME_MODE);
    if (fd < 0) return NULL;

    struct stat st;
    size_t sz = shm_frame_size(max_frame_size);
    if (fchmod(fd, SHM_FRAME_MODE) != 0 ||
        ftruncate(fd, (off_t)sz) != 0 ||
        fstat(fd, &st) != 0) {
        close(fd);
        shm_unlink(name);
        return NULL;
    }

    void *p = mmap(NULL, sz, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
    if (p == MAP_FAILED) {
        close(fd);
        shm_unlink(name);
        return NULL;
    }

    memset(p, 0, sz);
    shm_frame_ring_t *ring = (shm_frame_ring_t *)p;
    ring->magic          = SHM_FRAME_MAGIC;
    ring->version        = SHM_FRAME_VERSION;
    ring->max_frame_size = max_frame_size;

    shm_frame_handle_t *handle = shm_frame_alloc_handle(fd, sz, st.st_ino, p);
    if (!handle) {
        munmap(p, sz);
        close(fd);
        shm_unlink(name);
        return NULL;
    }
    return handle;
}

static inline int shm_frame_write(shm_frame_handle_t *handle,
                                  const uint8_t *data, uint32_t len,
                                  uint32_t width, uint32_t height,
                                  uint32_t format, uint8_t key_frame,
                                  uint64_t pts_us)
{
    if (!handle || !handle->ring || !data || len > handle->ring->max_frame_size) {
        return SHM_FRAME_WRITE_ERROR;
    }

    if (flock(handle->fd, LOCK_EX | LOCK_NB) < 0) {
        if (errno == EWOULDBLOCK || errno == EAGAIN) {
            return SHM_FRAME_WRITE_BUSY;
        }
        return SHM_FRAME_WRITE_ERROR;
    }

    shm_frame_ring_t *ring = handle->ring;
    const uint64_t next = __atomic_load_n(&ring->frame.frame_id, __ATOMIC_RELAXED) + 1;

    memcpy(ring->data, data, len);
    ring->frame.data_len  = len;
    ring->frame.width     = width;
    ring->frame.height    = height;
    ring->frame.format    = format;
    ring->frame.key_frame = key_frame;
    ring->frame.pts_us    = pts_us;

    __atomic_store_n(&ring->frame.frame_id, next, __ATOMIC_RELEASE);

    if (flock(handle->fd, LOCK_UN) < 0) {
        return SHM_FRAME_WRITE_ERROR;
    }
    return SHM_FRAME_WRITE_OK;
}

static inline void shm_frame_destroy(shm_frame_handle_t *handle, const char *name)
{
    if (handle) {
        if (handle->ring) munmap(handle->ring, handle->size);
        if (handle->fd >= 0) close(handle->fd);
        free(handle);
    }
    if (name) shm_unlink(name);
}

static inline shm_frame_handle_t *shm_frame_open(const char *name)
{
    int fd = shm_open(name, O_RDWR, 0);
    if (fd < 0) return NULL;

    if (fchmod(fd, SHM_FRAME_MODE) != 0) {
        close(fd);
        return NULL;
    }

    struct stat st;
    if (fstat(fd, &st) != 0 || (size_t)st.st_size < sizeof(shm_frame_ring_t)) {
        close(fd);
        return NULL;
    }

    void *p = mmap(NULL, (size_t)st.st_size, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
    if (p == MAP_FAILED) {
        close(fd);
        return NULL;
    }

    shm_frame_ring_t *ring = (shm_frame_ring_t *)p;
    if (ring->magic != SHM_FRAME_MAGIC ||
        ring->version != SHM_FRAME_VERSION ||
        (size_t)st.st_size < shm_frame_size(ring->max_frame_size)) {
        munmap(p, (size_t)st.st_size);
        close(fd);
        return NULL;
    }

    shm_frame_handle_t *handle = shm_frame_alloc_handle(fd, (size_t)st.st_size, st.st_ino, p);
    if (!handle) {
        munmap(p, (size_t)st.st_size);
        close(fd);
        return NULL;
    }
    return handle;
}

static inline void shm_frame_close(shm_frame_handle_t *handle)
{
    if (!handle) return;
    if (handle->ring) munmap(handle->ring, handle->size);
    if (handle->fd >= 0) close(handle->fd);
    free(handle);
}

#ifdef __cplusplus
}
#endif

#endif /* SHM_FRAME_H */
