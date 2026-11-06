// SPDX-License-Identifier: GPL-2.0
/* Erase and verify the LT7911EXC firmware region for recovery. */

#include <errno.h>
#include <fcntl.h>
#include <getopt.h>
#include <linux/i2c-dev.h>
#include <linux/i2c.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <unistd.h>

#define LT7911_PAGE_SELECT  0xff
#define LT7911_INFO_PAGE    0xe0
#define LT7911_FW_ADDR      0x00000
#define LT7911_FW_SIZE      (64 * 1024)
#define LT7911_FW_CRC_ADDR  (LT7911_FW_SIZE - 4)
#define LT7911_EMPTY_CRC    0x8f1805f1
#define LT7911_PAGE_SIZE    32
#define LT7911_POWER_PATH   "/proc/lt7911_info/power"
#define LT7911_CHIP_ID_PATH "/proc/lt7911_info/chip_id"
#define LT7911_DRIVER_NAME  "lt7911_manage"

struct options {
	int bus;
	uint16_t addr;
	int force;
};

struct lt7911 {
	int fd;
	uint16_t addr;
	int page;
};

static void usage(const char *prog)
{
	fprintf(stderr, "Usage: %s -b <bus> -a <addr> --force\n", prog);
}

static int parse_number(const char *text, unsigned long max,
			unsigned long *value)
{
	char *end;
	unsigned long parsed;

	errno = 0;
	parsed = strtoul(text, &end, 0);
	if (errno || !*text || *end || parsed > max)
		return -EINVAL;

	*value = parsed;
	return 0;
}

static int parse_args(int argc, char **argv, struct options *opts)
{
	static const struct option long_options[] = {
		{ "force", no_argument, NULL, 1 },
		{ "help", no_argument, NULL, 'h' },
		{ NULL, 0, NULL, 0 },
	};
	unsigned long value;
	int c;

	opts->bus = -1;
	opts->addr = 0x41;
	while ((c = getopt_long(argc, argv, "b:a:h", long_options, NULL)) != -1) {
		switch (c) {
		case 'b':
			if (parse_number(optarg, 0x7fffffff, &value))
				return -EINVAL;
			opts->bus = value;
			break;
		case 'a':
			if (parse_number(optarg, 0x7f, &value))
				return -EINVAL;
			opts->addr = value;
			break;
		case 1:
			opts->force = 1;
			break;
		case 'h':
			usage(argv[0]);
			exit(0);
		default:
			return -EINVAL;
		}
	}

	return opts->bus < 0 || !opts->force || optind != argc ? -EINVAL : 0;
}

static int write_text_file(const char *path, const char *text)
{
	ssize_t len = strlen(text);
	ssize_t written;
	int fd = open(path, O_WRONLY);

	if (fd < 0)
		return -errno;
	written = write(fd, text, len);
	if (written != len) {
		int ret = written < 0 ? -errno : -EIO;

		close(fd);
		return ret;
	}
	close(fd);
	return 0;
}

static int reset_device(void)
{
	int ret;

	ret = write_text_file(LT7911_POWER_PATH, "0");
	if (ret)
		return ret;
	ret = write_text_file(LT7911_POWER_PATH, "1");
	if (ret)
		write_text_file(LT7911_POWER_PATH, "1");
	return ret;
}

static int check_power_control(const struct options *opts)
{
	char path[128];
	char link[256];
	char chip_id[32];
	const char *name;
	ssize_t len;
	int fd;

	if (access(LT7911_POWER_PATH, W_OK))
		return -errno;

	snprintf(path, sizeof(path), "/sys/bus/i2c/devices/%d-%04x/driver",
		 opts->bus, opts->addr);
	len = readlink(path, link, sizeof(link) - 1);
	if (len < 0)
		return -errno;
	link[len] = '\0';
	name = strrchr(link, '/');
	if (!name || strcmp(name + 1, LT7911_DRIVER_NAME))
		return -ENODEV;

	fd = open(LT7911_CHIP_ID_PATH, O_RDONLY);
	if (fd < 0)
		return -errno;
	len = read(fd, chip_id, sizeof(chip_id) - 1);
	if (len < 0) {
		int ret = -errno;

		close(fd);
		return ret;
	}
	close(fd);
	chip_id[len] = '\0';
	return strcmp(chip_id, "lt7911exc\n") ? -ENODEV : 0;
}

static int transfer(struct lt7911 *lt, struct i2c_msg *msgs, int count)
{
	struct i2c_rdwr_ioctl_data data = {
		.msgs = msgs,
		.nmsgs = count,
	};
	int ret;

	ret = ioctl(lt->fd, I2C_RDWR, &data);
	if (ret < 0)
		return -errno;
	return ret == count ? 0 : -EIO;
}

static int write_buf(struct lt7911 *lt, uint8_t *buf, uint16_t len)
{
	struct i2c_msg msg = {
		.addr = lt->addr,
		.len = len,
		.buf = buf,
	};

	return transfer(lt, &msg, 1);
}

static int read_page_selector(struct lt7911 *lt, uint8_t *page)
{
	uint8_t reg = LT7911_PAGE_SELECT;
	struct i2c_msg msgs[2];

	memset(msgs, 0, sizeof(msgs));
	msgs[0].addr = lt->addr;
	msgs[0].len = 1;
	msgs[0].buf = &reg;
	msgs[1].addr = lt->addr;
	msgs[1].flags = I2C_M_RD;
	msgs[1].len = 1;
	msgs[1].buf = page;
	return transfer(lt, msgs, 2);
}

static int select_page(struct lt7911 *lt, uint8_t page)
{
	uint8_t page_buf[2] = { LT7911_PAGE_SELECT, page };
	uint8_t current;
	int ret;

	ret = read_page_selector(lt, &current);
	if (ret)
		return ret;
	lt->page = current;
	if (current == page)
		return 0;
	ret = write_buf(lt, page_buf, sizeof(page_buf));
	if (!ret)
		lt->page = page;
	return ret;
}

static int write_reg(struct lt7911 *lt, uint8_t reg, uint8_t value)
{
	uint8_t reg_buf[2] = { reg, value };
	int ret = select_page(lt, LT7911_INFO_PAGE);

	return ret ? ret : write_buf(lt, reg_buf, sizeof(reg_buf));
}

static int bulk_read(struct lt7911 *lt, uint8_t page, uint8_t reg,
		     uint8_t *data, uint16_t len)
{
	struct i2c_msg msgs[2];
	int ret = select_page(lt, page);

	if (ret)
		return ret;
	memset(msgs, 0, sizeof(msgs));
	msgs[0].addr = lt->addr;
	msgs[0].len = 1;
	msgs[0].buf = &reg;
	msgs[1].addr = lt->addr;
	msgs[1].flags = I2C_M_RD;
	msgs[1].len = len;
	msgs[1].buf = data;
	return transfer(lt, msgs, 2);
}

static int open_i2c(struct lt7911 *lt, const struct options *opts)
{
	char path[64];
	int request = opts->force ? I2C_SLAVE_FORCE : I2C_SLAVE;

	snprintf(path, sizeof(path), "/dev/i2c-%d", opts->bus);
	lt->fd = open(path, O_RDWR);
	if (lt->fd < 0)
		return -errno;
	lt->addr = opts->addr;
	lt->page = -1;
	if (ioctl(lt->fd, request, opts->addr) < 0) {
		int ret = -errno;

		close(lt->fd);
		lt->fd = -1;
		return ret;
	}
	return 0;
}

static int erase_block(struct lt7911 *lt, uint32_t addr)
{
	const uint8_t sequence[][2] = {
		{ 0xee, 0x01 },
		{ 0x54, 0x01 },
		{ 0x55, 0x06 },
		{ 0x51, 0x01 },
		{ 0x51, 0x00 },
		{ 0x54, 0x05 },
		{ 0x55, 0xd8 },
		{ 0x5a, (addr >> 16) & 0xff },
		{ 0x5b, (addr >> 8) & 0xff },
		{ 0x5c, addr & 0xff },
		{ 0x51, 0x01 },
		{ 0x51, 0x00 },
	};
	size_t i;
	int ret;

	for (i = 0; i < sizeof(sequence) / sizeof(sequence[0]); i++) {
		ret = write_reg(lt, sequence[i][0], sequence[i][1]);
		if (ret)
			return ret;
	}
	usleep(200000);
	return 0;
}

static int reset_fifo(struct lt7911 *lt)
{
	int ret;

	ret = write_reg(lt, 0x5f, 0x08);
	if (!ret)
		ret = write_reg(lt, 0x5f, 0x00);
	if (ret)
		return ret;
	usleep(1000000);
	return 0;
}

static int read_flash_page(struct lt7911 *lt, uint32_t addr, uint8_t *data,
			   uint8_t len)
{
	int ret;

	ret = write_reg(lt, 0x54, 0x45);
	if (!ret) ret = write_reg(lt, 0x55, 0x03);
	if (!ret) ret = write_reg(lt, 0x56, len);
	if (!ret) ret = write_reg(lt, 0x57, 0x00);
	if (!ret) ret = write_reg(lt, 0x58, 0x00);
	if (!ret) ret = write_reg(lt, 0x5a, addr >> 16);
	if (!ret) ret = write_reg(lt, 0x5b, addr >> 8);
	if (!ret) ret = write_reg(lt, 0x5c, addr);
	if (!ret) ret = write_reg(lt, 0x51, 0x01);
	if (!ret) ret = write_reg(lt, 0x51, 0x00);
	if (!ret) ret = bulk_read(lt, LT7911_INFO_PAGE, 0x5e, data, len);
	return ret;
}

static int verify_erased_range(struct lt7911 *lt, uint32_t addr, size_t len)
{
	uint8_t readback[LT7911_PAGE_SIZE];
	size_t offset;
	int ret;

	ret = reset_fifo(lt);
	if (ret)
		return ret;

	for (offset = 0; offset < len; offset += LT7911_PAGE_SIZE) {
		size_t page_len = len - offset;
		size_t i;

		if (page_len > LT7911_PAGE_SIZE)
			page_len = LT7911_PAGE_SIZE;
		ret = read_flash_page(lt, addr + offset, readback,
				      page_len);
		if (ret)
			return ret;
		for (i = 0; i < page_len; i++) {
			if (readback[i] != 0xff) {
				fprintf(stderr,
					"erase mismatch at 0x%05x: expected 0xff, actual 0x%02x\n",
					addr + (unsigned int)offset +
					(unsigned int)i, readback[i]);
				return -EIO;
			}
		}
	}
	return 0;
}

static int verify_firmware_erased(struct lt7911 *lt)
{
	uint8_t crc[4];
	uint32_t actual;
	int ret;

	ret = write_reg(lt, 0xee, 0x01);
	if (!ret) ret = write_reg(lt, 0x7b, 0x60);
	if (!ret) ret = write_reg(lt, 0x7b, 0x40);
	if (ret)
		return ret;
	usleep(150000);

	ret = bulk_read(lt, 0x00, 0x22, crc, sizeof(crc));
	if (ret)
		return ret;
	actual = ((uint32_t)crc[0] << 24) |
		 ((uint32_t)crc[1] << 16) |
		 ((uint32_t)crc[2] << 8) | crc[3];
	printf("Erased firmware CRC: expected 0x%08x, actual 0x%08x\n",
	       LT7911_EMPTY_CRC, actual);
	if (actual != LT7911_EMPTY_CRC)
		return -EIO;

	return verify_erased_range(lt, LT7911_FW_CRC_ADDR, 4);
}

int main(int argc, char **argv)
{
	struct options opts = { 0 };
	struct lt7911 lt = { .fd = -1 };
	int unlock_ret;
	int ret;

	setvbuf(stdout, NULL, _IOLBF, 0);

	ret = parse_args(argc, argv, &opts);
	if (ret) {
		usage(argv[0]);
		return 2;
	}
	ret = check_power_control(&opts);
	if (ret) {
		fprintf(stderr, "power control is unavailable: %s (%d)\n",
			strerror(-ret), ret);
		return 1;
	}

	ret = open_i2c(&lt, &opts);
	if (ret)
		goto out;

	printf("Erasing firmware block at 0x%05x...\n", LT7911_FW_ADDR);
	ret = erase_block(&lt, LT7911_FW_ADDR);
	if (ret)
		goto out_unlock;
	printf("Verifying erased firmware with hardware CRC...\n");
	ret = verify_firmware_erased(&lt);
	if (ret)
		goto out_unlock;

out_unlock:
	unlock_ret = write_reg(&lt, 0xee, 0x00);
	if (!ret)
		ret = unlock_ret;
	close(lt.fd);
	lt.fd = -1;
	if (!ret)
		ret = reset_device();
out:
	if (lt.fd >= 0)
		close(lt.fd);
	if (ret) {
		fprintf(stderr, "firmware erase failed: %s (%d)\n",
			strerror(-ret), ret);
		return 1;
	}

	printf("LT7911EXC firmware erased, verified, and reset through %s; EDID preserved\n",
	       LT7911_POWER_PATH);
	return 0;
}
