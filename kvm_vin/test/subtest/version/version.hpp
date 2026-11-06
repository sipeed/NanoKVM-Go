#ifndef __KVM_UI_TEST_SUBTEST_VERSION_HPP__
#define __KVM_UI_TEST_SUBTEST_VERSION_HPP__

#include <buildinfo/version.hpp>
#include <fmt/base.h>
#include <fmt/core.h>

void version()
{
    fmt::println(
        "GIT_HASH={} GIT_DIRTY={} {}", GIT_HASH, GIT_DIRTY, BUILD_TIMESTAMP);
}

#endif  // __KVM_UI_TEST_SUBTEST_VERSION_HPP__
