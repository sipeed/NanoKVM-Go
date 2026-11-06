#include <catch2/catch_test_macros.hpp>

#include <chrono>

#include "lvgl_main/auto_rotation_runtime.hpp"

using lvgl_main::AutoRotationConfirmation;
using lvgl_main::ImuOrientation;
using lvgl_main::ImuSample;
using lvgl_main::classify_imu_orientation;
using lvgl_main::remaining_auto_rotation_sleep;

TEST_CASE("auto rotation classifies dominant x-axis orientation")
{
	CHECK(classify_imu_orientation(
		      ImuSample { .x_raw = 501, .y_raw = 100, .z_raw = 100 }) ==
	      ImuOrientation::Forward);
	CHECK(classify_imu_orientation(ImuSample {
		      .x_raw = -501, .y_raw = 100, .z_raw = 100 }) ==
	      ImuOrientation::Reverse);
}

TEST_CASE("auto rotation rejects weak or non-dominant x-axis samples")
{
	CHECK(classify_imu_orientation(
		      ImuSample { .x_raw = 500, .y_raw = 0, .z_raw = 0 }) ==
	      ImuOrientation::Unknown);
	CHECK(classify_imu_orientation(
		      ImuSample { .x_raw = 700, .y_raw = 500, .z_raw = 0 }) ==
	      ImuOrientation::Unknown);
}

TEST_CASE("auto rotation confirmation requires consecutive changed samples")
{
	AutoRotationConfirmation confirmation(3, false);

	CHECK_FALSE(confirmation.update(ImuOrientation::Reverse).has_value());
	CHECK_FALSE(confirmation.update(ImuOrientation::Reverse).has_value());
	const auto reverse = confirmation.update(ImuOrientation::Reverse);
	REQUIRE(reverse.has_value());
	CHECK(*reverse);

	CHECK_FALSE(confirmation.update(ImuOrientation::Reverse).has_value());
	CHECK_FALSE(confirmation.update(ImuOrientation::Forward).has_value());
	CHECK_FALSE(confirmation.update(ImuOrientation::Forward).has_value());
	const auto forward = confirmation.update(ImuOrientation::Forward);
	REQUIRE(forward.has_value());
	CHECK_FALSE(*forward);
}

TEST_CASE("auto rotation confirmation resets on unknown samples")
{
	AutoRotationConfirmation confirmation(2, false);

	CHECK_FALSE(confirmation.update(ImuOrientation::Reverse).has_value());
	CHECK_FALSE(confirmation.update(ImuOrientation::Unknown).has_value());
	CHECK_FALSE(confirmation.update(ImuOrientation::Reverse).has_value());
	const auto reverse = confirmation.update(ImuOrientation::Reverse);
	REQUIRE(reverse.has_value());
	CHECK(*reverse);
}

TEST_CASE("auto rotation pacing subtracts elapsed read time")
{
	using namespace std::chrono_literals;

	CHECK(remaining_auto_rotation_sleep(125ms, 0ms) == 125ms);
	CHECK(remaining_auto_rotation_sleep(125ms, 30ms) == 95ms);
	CHECK(remaining_auto_rotation_sleep(125ms, 125ms) == 0ms);
	CHECK(remaining_auto_rotation_sleep(125ms, 200ms) == 0ms);
	CHECK(remaining_auto_rotation_sleep(0ms, 0ms) == 0ms);
}
