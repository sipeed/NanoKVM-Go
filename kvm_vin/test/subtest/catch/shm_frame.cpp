#include <algorithm>
#include <array>
#include <atomic>
#include <cstdint>
#include <string>
#include <utility>

#include <catch2/catch_test_macros.hpp>
#include <fcntl.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <unistd.h>

#include "shm_frame.h"

namespace {

std::string unique_shm_name()
{
    static std::atomic<unsigned int> sequence {0};
    return "/nanokvm_shm_frame_test_" + std::to_string(getpid()) + "_"
        + std::to_string(sequence.fetch_add(1));
}

class ShmCleanup {
  public:
    explicit ShmCleanup(std::string name)
        : name_(std::move(name))
    {
        (void)shm_unlink(name_.c_str());
    }

    ~ShmCleanup() { (void)shm_unlink(name_.c_str()); }

    const char* name() const { return name_.c_str(); }

  private:
    std::string name_;
};

mode_t descriptor_mode(int fd)
{
    struct stat st {};
    if (fstat(fd, &st) != 0) {
        return static_cast<mode_t>(-1);
    }
    return st.st_mode & 0777;
}

}  // namespace

TEST_CASE("shared frame creation enforces owner-only mode and preserves frame flow",
          "[shm_frame][security]")
{
    CHECK((SHM_FRAME_CREATE_FLAGS & O_EXCL) != 0);
    ShmCleanup cleanup(unique_shm_name());
    const mode_t previous_umask = umask(0);
    shm_frame_handle_t* writer = shm_frame_create(cleanup.name(), 4096);
    (void)umask(previous_umask);

    REQUIRE(writer != nullptr);
    CHECK(descriptor_mode(writer->fd) == SHM_FRAME_MODE);

    const std::array<uint8_t, 5> payload {0x00, 0x01, 0x02, 0x03, 0x04};
    REQUIRE(shm_frame_write(writer,
                            payload.data(),
                            static_cast<uint32_t>(payload.size()),
                            1920,
                            1080,
                            0x34363248,
                            1,
                            123456)
            == SHM_FRAME_WRITE_OK);

    int reader_fd = shm_open(cleanup.name(), O_RDONLY, 0);
    REQUIRE(reader_fd >= 0);
    CHECK(descriptor_mode(reader_fd) == SHM_FRAME_MODE);

    const void* mapping = mmap(nullptr, writer->size, PROT_READ, MAP_SHARED, reader_fd, 0);
    REQUIRE(mapping != MAP_FAILED);
    const auto* ring = static_cast<const shm_frame_ring_t*>(mapping);
    CHECK(ring->frame.data_len == payload.size());
    CHECK(ring->frame.width == 1920);
    CHECK(ring->frame.height == 1080);
    CHECK(ring->frame.frame_id == 1);
    CHECK(std::equal(payload.begin(), payload.end(), ring->data));

    REQUIRE(munmap(const_cast<void*>(mapping), writer->size) == 0);
    REQUIRE(close(reader_fd) == 0);
    shm_frame_destroy(writer, cleanup.name());
}

TEST_CASE("shared frame creation replaces a permissive legacy object privately",
          "[shm_frame][security]")
{
    ShmCleanup cleanup(unique_shm_name());
    const mode_t previous_umask = umask(0);
    int legacy_fd = shm_open(cleanup.name(), O_CREAT | O_EXCL | O_RDWR, 0666);
    (void)umask(previous_umask);

    REQUIRE(legacy_fd >= 0);
    REQUIRE(ftruncate(legacy_fd, 128) == 0);
    CHECK(descriptor_mode(legacy_fd) == 0666);

    struct stat legacy_stat {};
    REQUIRE(fstat(legacy_fd, &legacy_stat) == 0);

    shm_frame_handle_t* replacement = shm_frame_create(cleanup.name(), 4096);
    REQUIRE(replacement != nullptr);
    CHECK(descriptor_mode(replacement->fd) == SHM_FRAME_MODE);
    CHECK(replacement->ino != legacy_stat.st_ino);

    shm_frame_handle_t* consumer = shm_frame_open(cleanup.name());
    REQUIRE(consumer != nullptr);
    CHECK(descriptor_mode(consumer->fd) == SHM_FRAME_MODE);

    shm_frame_close(consumer);
    shm_frame_destroy(replacement, cleanup.name());
    REQUIRE(close(legacy_fd) == 0);
}

TEST_CASE("shared frame open repairs a permissive valid object in place",
          "[shm_frame][security]")
{
    ShmCleanup cleanup(unique_shm_name());
    shm_frame_handle_t* writer = shm_frame_create(cleanup.name(), 4096);
    REQUIRE(writer != nullptr);
    const ino_t inode = writer->ino;
    REQUIRE(fchmod(writer->fd, 0666) == 0);
    CHECK(descriptor_mode(writer->fd) == 0666);
    shm_frame_close(writer);

    shm_frame_handle_t* consumer = shm_frame_open(cleanup.name());
    REQUIRE(consumer != nullptr);
    CHECK(consumer->ino == inode);
    CHECK(descriptor_mode(consumer->fd) == SHM_FRAME_MODE);

    shm_frame_close(consumer);
}
