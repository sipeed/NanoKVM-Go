#include <catch2/catch_test_macros.hpp>

#if NANOKVM_GO_SIMULATOR_BUILD
#include "lvgl_main/simulator_build_config.hpp"
#endif

TEST_CASE("simulator display configuration fixes the logical canvas and zoom",
	  "[simulator][display]")
{
#if NANOKVM_GO_SIMULATOR_BUILD
	CHECK(lvgl_main::simulator_build_config::kLogicalWidth == 270);
	CHECK(lvgl_main::simulator_build_config::kLogicalHeight == 240);
	CHECK(lvgl_main::simulator_build_config::kWindowZoom == 2);
#else
	SUCCEED("The simulator display configuration is not selected");
#endif
}
