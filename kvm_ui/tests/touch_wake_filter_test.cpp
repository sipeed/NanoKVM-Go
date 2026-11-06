#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/touch_wake_filter.hpp"

using lvgl_main::TouchWakeFilter;

TEST_CASE("touch wake filter passes normal screen-on touches")
{
	TouchWakeFilter filter;

	CHECK_FALSE(filter.should_suppress(true, true, true));
	CHECK_FALSE(filter.should_suppress(true, false, true));
	CHECK_FALSE(filter.should_suppress(true, true, false));
}

TEST_CASE("touch wake filter suppresses the first screen-off press")
{
	TouchWakeFilter filter;

	CHECK(filter.should_suppress(false, true, true));
	CHECK(filter.should_suppress(false, false, true));
	CHECK_FALSE(filter.should_suppress(false, true, false));
}

TEST_CASE("touch wake filter holds suppression after the screen wakes")
{
	TouchWakeFilter filter;

	CHECK(filter.should_suppress(false, true, true));
	CHECK(filter.should_suppress(true, false, true));
	CHECK_FALSE(filter.should_suppress(true, true, false));
	CHECK_FALSE(filter.should_suppress(true, true, true));
}

TEST_CASE("touch wake filter lets the next touch through after release")
{
	TouchWakeFilter filter;

	CHECK(filter.should_suppress(false, true, true));
	CHECK_FALSE(filter.should_suppress(false, true, false));
	CHECK_FALSE(filter.should_suppress(false, true, true));
}

TEST_CASE("touch wake filter reports released for a quick wake tap")
{
	TouchWakeFilter filter;

	CHECK_FALSE(filter.should_suppress(false, true, false));
	CHECK_FALSE(filter.should_suppress(false, true, true));
}

TEST_CASE("touch wake filter rearms for a later sleep cycle")
{
	TouchWakeFilter filter;

	CHECK(filter.should_suppress(false, true, true));
	CHECK_FALSE(filter.should_suppress(false, true, false));
	CHECK_FALSE(filter.should_suppress(true, false, false));
	CHECK(filter.should_suppress(false, true, true));
}
