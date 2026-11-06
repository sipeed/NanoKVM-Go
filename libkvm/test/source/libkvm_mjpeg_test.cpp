#include <catch2/catch_test_macros.hpp>

#include "libkvm/kvm_vision.h"

TEST_CASE("mjpeg device info", "[library]")
{
    kvmv_init(true);
    kvmv_deinit();
}
