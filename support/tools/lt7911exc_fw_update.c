// SPDX-License-Identifier: GPL-2.0
/*
 * LT7911EXC firmware updater.
 *
 * Keep the flash programming and CRC sequence aligned with
 * drivers/misc/lt7911_manage.c:
 * erase firmware block, program 32-byte pages, write the padded-image CRC,
 * optionally reset the device as requested, then verify the hardware CRC.
 */

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
#include <sys/stat.h>
#include <unistd.h>

#define LT7911EXC_FW_SIZE       (64 * 1024)
#define LT7911EXC_FW_DATA_SIZE  (LT7911EXC_FW_SIZE - 4)
#define LT7911EXC_PAGE_SIZE     32
#define LT7911EXC_INFO_PAGE     0xe0
#define LT7911_PAGE_SELECT      0xff
#define LT7911_POWER_PATH       "/proc/lt7911_info/power"
#define LT7911_CHIP_ID_PATH     "/proc/lt7911_info/chip_id"
#define LT7911_DRIVER_NAME      "lt7911_manage"

struct lt7911 {
	int fd;
	uint16_t addr;
	int page;
};

struct options {
	const char *firmware;
	int bus;
	uint16_t addr;
	int force;
	int reset;
};

static void usage(const char *prog)
{
	fprintf(stderr,
		"Usage: %s -b <bus> [-a <addr>] --force -f <fw.bin> "
		"-r <0|1>\n"
		"  -a <addr>           I2C address (default: 0x41)\n"
		"  -r, --reset <0|1>  reset after update (1: reset, 0: skip)\n",
		prog);
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
		{ "reset", required_argument, NULL, 'r' },
		{ "help", no_argument, NULL, 'h' },
		{ NULL, 0, NULL, 0 },
	};
	unsigned long value;
	int c;

	opts->bus = -1;
	opts->addr = 0x41;
	opts->reset = -1;
	while ((c = getopt_long(argc, argv, "b:a:f:r:h", long_options, NULL)) != -1) {
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
		case 'f':
			opts->firmware = optarg;
			break;
		case 'r':
			if (parse_number(optarg, 1, &value))
				return -EINVAL;
			opts->reset = value;
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

	return opts->bus < 0 || !opts->firmware || !opts->force ||
		opts->reset < 0 || optind != argc ? -EINVAL : 0;
}

static int write_text_file(const char *path, const char *text)
{
	ssize_t len = strlen(text);
	ssize_t written;
	int fd;

	fd = open(path, O_WRONLY);
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

static int check_device(const struct options *opts)
{
	char path[128];
	char link[256];
	char chip_id[32];
	const char *name;
	ssize_t len;
	int fd;

	if (opts->reset && access(LT7911_POWER_PATH, W_OK))
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

static int i2c_transfer(struct lt7911 *lt, struct i2c_msg *msgs, int count)
{
	struct i2c_rdwr_ioctl_data transfer = {
		.msgs = msgs,
		.nmsgs = count,
	};
	int ret = ioctl(lt->fd, I2C_RDWR, &transfer);

	if (ret < 0)
		return -errno;
	return ret == count ? 0 : -EIO;
}

static int i2c_write(struct lt7911 *lt, uint8_t *buf, uint16_t len)
{
	struct i2c_msg msg = {
		.addr = lt->addr,
		.len = len,
		.buf = buf,
	};

	return i2c_transfer(lt, &msg, 1);
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
	return i2c_transfer(lt, msgs, 2);
}

static int select_page(struct lt7911 *lt, uint8_t page)
{
	uint8_t buf[2] = { LT7911_PAGE_SELECT, page };
	uint8_t current;
	int ret;

	ret = read_page_selector(lt, &current);
	if (ret)
		return ret;
	lt->page = current;
	if (current == page)
		return 0;
	ret = i2c_write(lt, buf, sizeof(buf));
	if (!ret)
		lt->page = page;
	return ret;
}

static int write_reg(struct lt7911 *lt, uint8_t page, uint8_t reg,
		     uint8_t value)
{
	uint8_t buf[2] = { reg, value };
	int ret = select_page(lt, page);

	return ret ? ret : i2c_write(lt, buf, sizeof(buf));
}

static int raw_write(struct lt7911 *lt, uint8_t page, uint8_t reg,
		     const uint8_t *data, size_t len)
{
	uint8_t buf[1 + LT7911EXC_PAGE_SIZE];
	int ret;

	if (!len || len > LT7911EXC_PAGE_SIZE)
		return -EINVAL;
	ret = select_page(lt, page);
	if (ret)
		return ret;
	buf[0] = reg;
	memcpy(buf + 1, data, len);
	return i2c_write(lt, buf, len + 1);
}

static int bulk_read(struct lt7911 *lt, uint8_t page, uint8_t reg,
		     uint8_t *data, size_t len)
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
	return i2c_transfer(lt, msgs, 2);
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

static int read_firmware(const char *path, uint8_t **data, size_t *size)
{
	struct stat st;
	uint8_t *buf;
	size_t offset = 0;
	int fd = open(path, O_RDONLY);

	if (fd < 0)
		return -errno;
	if (fstat(fd, &st) || st.st_size <= 0 ||
	    st.st_size > LT7911EXC_FW_DATA_SIZE) {
		int ret = errno ? -errno : -EINVAL;

		close(fd);
		return ret;
	}
	buf = malloc(st.st_size);
	if (!buf) {
		close(fd);
		return -ENOMEM;
	}
	while (offset < (size_t)st.st_size) {
		ssize_t ret = read(fd, buf + offset, st.st_size - offset);

		if (ret <= 0) {
			int error = ret < 0 ? -errno : -EIO;

			free(buf);
			close(fd);
			return error;
		}
		offset += ret;
	}
	close(fd);
	*data = buf;
	*size = st.st_size;
	return 0;
}

static uint32_t crc_byte(uint32_t crc, uint8_t data)
{
	int bit;

	crc ^= (uint32_t)data << 24;
	for (bit = 0; bit < 8; bit++)
		crc = crc & 0x80000000 ?
			(crc << 1) ^ 0x04c11db7 : crc << 1;
	return crc;
}

static uint32_t firmware_crc(const uint8_t *data)
{
	uint32_t crc = 0xffffffff;
	size_t i;

	for (i = 0; i < LT7911EXC_FW_DATA_SIZE; i += 4) {
		crc = crc_byte(crc, data[i + 3]);
		crc = crc_byte(crc, data[i + 2]);
		crc = crc_byte(crc, data[i + 1]);
		crc = crc_byte(crc, data[i]);
	}
	return crc;
}

static int erase_firmware(struct lt7911 *lt)
{
	static const uint8_t sequence[][2] = {
		{ 0xee, 0x01 }, { 0x54, 0x01 }, { 0x55, 0x06 },
		{ 0x51, 0x01 }, { 0x51, 0x00 }, { 0x54, 0x05 },
		{ 0x55, 0xd8 }, { 0x5a, 0x00 }, { 0x5b, 0x00 },
		{ 0x5c, 0x00 }, { 0x51, 0x01 }, { 0x51, 0x00 },
	};
	size_t i;
	int ret;

	for (i = 0; i < sizeof(sequence) / sizeof(sequence[0]); i++) {
		ret = write_reg(lt, LT7911EXC_INFO_PAGE, sequence[i][0],
				sequence[i][1]);
		if (ret)
			return ret;
	}
	usleep(200000);
	return 0;
}

static int program_init(struct lt7911 *lt, uint32_t addr)
{
	int ret;

	ret = write_reg(lt, 0xe0, 0xee, 0x01);
	if (!ret) ret = write_reg(lt, 0xe0, 0x5f, 0x01);
	if (!ret) ret = write_reg(lt, 0xe0, 0x5a, addr >> 16);
	if (!ret) ret = write_reg(lt, 0xe0, 0x5b, addr >> 8);
	if (!ret) ret = write_reg(lt, 0xe0, 0x5c, addr);
	return ret;
}

static int program_data(struct lt7911 *lt, const uint8_t *data, size_t len)
{
	uint32_t addr = 0;
	size_t offset;
	int ret;

	for (offset = 0; offset < len; offset += LT7911EXC_PAGE_SIZE) {
		size_t page_len = len - offset;

		if (page_len > LT7911EXC_PAGE_SIZE)
			page_len = LT7911EXC_PAGE_SIZE;
		ret = program_init(lt, addr);
		if (!ret)
			ret = raw_write(lt, 0xe0, 0x5d, data + offset, page_len);
		if (!ret && page_len < LT7911EXC_PAGE_SIZE)
			ret = write_reg(lt, 0xe0, 0x5f, 0x05);
		if (!ret && page_len < LT7911EXC_PAGE_SIZE) {
			ret = write_reg(lt, 0xe0, 0x5f, 0x01);
			usleep(1500);
		}
		if (!ret)
			ret = write_reg(lt, 0xe0, 0x5f, 0x00);
		if (ret)
			return ret;
		addr += LT7911EXC_PAGE_SIZE;
		if (!(offset % 4096) || offset + page_len == len) {
			printf("\rprogramming: %zu/%zu bytes", offset + page_len, len);
			fflush(stdout);
		}
	}
	putchar('\n');
	return 0;
}

static int write_crc(struct lt7911 *lt, uint32_t value)
{
	const uint32_t addr = LT7911EXC_FW_SIZE - 4;
	uint8_t crc[4] = { value, value >> 8, value >> 16, value >> 24 };
	int ret = program_init(lt, addr);

	if (!ret) ret = raw_write(lt, 0xe0, 0x5d, crc, sizeof(crc));
	if (!ret) ret = write_reg(lt, 0xe0, 0x5f, 0x05);
	if (!ret) ret = write_reg(lt, 0xe0, 0x5f, 0x01);
	if (!ret) usleep(1500);
	if (!ret) ret = write_reg(lt, 0xe0, 0x5f, 0x00);
	return ret;
}

static int verify_crc(struct lt7911 *lt, uint32_t expected_crc,
		      uint32_t *actual_crc)
{
	uint8_t crc[4];
	int ret;

	ret = write_reg(lt, 0xe0, 0xee, 0x01);
	if (!ret) ret = write_reg(lt, 0xe0, 0x7b, 0x60);
	if (!ret) ret = write_reg(lt, 0xe0, 0x7b, 0x40);
	if (ret)
		return ret;
	usleep(150000);

	ret = bulk_read(lt, 0x00, 0x22, crc, sizeof(crc));
	if (ret)
		return ret;
	*actual_crc = ((uint32_t)crc[0] << 24) |
		      ((uint32_t)crc[1] << 16) |
		      ((uint32_t)crc[2] << 8) | crc[3];
	printf("Hardware CRC: expected 0x%08x, actual 0x%08x\n",
	       expected_crc, *actual_crc);
	return *actual_crc == expected_crc ? 0 : -EIO;
}

int main(int argc, char **argv)
{
	struct options opts = { 0 };
	struct lt7911 lt = { .fd = -1 };
	uint8_t *firmware = NULL;
	uint8_t *crc_data = NULL;
	uint32_t crc;
	uint32_t actual_crc = 0;
	size_t firmware_size = 0;
	int ret;

	ret = parse_args(argc, argv, &opts);
	if (ret) {
		usage(argv[0]);
		return 2;
	}
	ret = check_device(&opts);
	if (ret) {
		fprintf(stderr, "LT7911EXC device check failed: %s (%d)\n",
			strerror(-ret), ret);
		goto out_error;
	}
	ret = read_firmware(opts.firmware, &firmware, &firmware_size);
	if (ret)
		goto out_error;
	crc_data = malloc(LT7911EXC_FW_DATA_SIZE);
	if (!crc_data) {
		ret = -ENOMEM;
		goto out_error;
	}
	memset(crc_data, 0xff, LT7911EXC_FW_DATA_SIZE);
	memcpy(crc_data, firmware, firmware_size);
	crc = firmware_crc(crc_data);
	free(crc_data);
	crc_data = NULL;

	ret = open_i2c(&lt, &opts);
	if (!ret) ret = erase_firmware(&lt);
	if (!ret) ret = program_data(&lt, firmware, firmware_size);
	if (!ret) ret = write_crc(&lt, crc);
	if (ret)
		goto out_unlock;

	if (opts.reset) {
		ret = reset_device();
		lt.page = -1;
		if (ret) {
			fprintf(stderr,
				"firmware written, but GPIO reset failed: %s\n",
				strerror(-ret));
			goto out_unlock;
		}
	}
	printf("Verifying firmware with hardware CRC...\n");
	ret = verify_crc(&lt, crc, &actual_crc);

out_unlock:
	if (lt.fd >= 0) {
		int unlock_ret = write_reg(&lt, 0xe0, 0xee, 0x00);

		if (!ret)
			ret = unlock_ret;
		close(lt.fd);
		lt.fd = -1;
	}
	if (ret)
		goto out_error;
	if (opts.reset) {
		printf("LT7911EXC firmware update and reset completed\n");
	} else {
		printf("LT7911EXC firmware update completed; reset skipped, "
		       "external reset or power cycle required to load new firmware\n");
	}
	free(firmware);
	return 0;

out_error:
	if (lt.fd >= 0)
		close(lt.fd);
	free(crc_data);
	free(firmware);
	if (ret)
		fprintf(stderr, "failed: %s (%d)\n", strerror(-ret), ret);
	return 1;
}
