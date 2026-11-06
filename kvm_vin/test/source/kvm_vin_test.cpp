// Catch2
#define CATCH_CONFIG_RUNNER
// Catch2 include
#include <catch2/catch_all.hpp>
#include <catch2/catch_session.hpp>
#include <catch2/catch_test_macros.hpp>
#include <tools/segfault_stack_tracer.h>

//
#include "../subtest/version/version.hpp"

TEST_CASE("Catch2 Test from main.cpp", "[Catch2][main]") { REQUIRE(true); }

int main(int argc, char* argv[])
{
    tools::SegfaultStackTracker::init();
    version();

    Catch::Session session;
    int ret = session.applyCommandLine(argc, argv);
    if (ret != 0) {
        return ret;
    }

    return session.run();
}
