#include <catch2/catch_session.hpp>
#include <tools/segfault_stack_tracer.h>

#include "../subtest/version/version.hpp"

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
