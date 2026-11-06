#include <chrono>
#include <optional>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/ssid_raster_install_gate.hpp"
#include "lvgl_main/ssid_raster_result_mailbox.hpp"

using namespace lvgl_main;

namespace
{

ui::SsidPreparedRasterResult result_for(std::uint64_t generation)
{
	ui::SsidPreparedRasterResult result;
	result.identity.page_token = 7;
	result.identity.generation = generation;
	result.identity.font_generation = 1;
	result.identity.viewport_width = 150;
	result.identity.height = 32;
	result.identity.font_size_px = 20;
	result.identity.byte_budget = 4096;
	result.identity.display_text = "redacted";
	return result;
}

} // namespace

TEST_CASE("SSID raster mailbox retains only the latest completed result")
{
	SsidRasterResultMailbox mailbox;

	REQUIRE(mailbox.offer(result_for(1)));
	REQUIRE(mailbox.offer(result_for(3)));

	const auto result = mailbox.take();
	REQUIRE(result.has_value());
	CHECK(result->identity.generation == 3);
	CHECK_FALSE(mailbox.take().has_value());
}

TEST_CASE("SSID raster mailbox drops pending data on close")
{
	SsidRasterResultMailbox mailbox;

	REQUIRE(mailbox.offer(result_for(1)));
	mailbox.close();

	CHECK_FALSE(mailbox.take().has_value());
	CHECK_FALSE(mailbox.offer(result_for(2)));
}

TEST_CASE("SSID raster mailbox retains its latest result until input is quiet")
{
	SsidRasterResultMailbox mailbox;
	SsidRasterInstallGate gate;
	const auto started = SsidRasterInstallGate::TimePoint {};
	auto take_if_input_quiet = [&mailbox, &gate](bool touch_pressed,
						      SsidRasterInstallGate::TimePoint now) {
		if (gate.blocks_install(touch_pressed, now)) {
			return std::optional<ui::SsidPreparedRasterResult> {};
		}
		return mailbox.take();
	};

	gate.reset(10U, started);
	gate.observe(11U, started);
	CHECK(gate.blocks_install(false, started));
	CHECK(gate.blocks_install(false, started + std::chrono::milliseconds(499)));
	CHECK_FALSE(
		gate.blocks_install(false, started + std::chrono::milliseconds(500)));

	REQUIRE(mailbox.offer(result_for(1U)));
	REQUIRE(mailbox.offer(result_for(2U)));
	CHECK_FALSE(take_if_input_quiet(true,
		started + std::chrono::milliseconds(500)).has_value());
	// The one-slot mailbox remains occupied while input is held, so the worker
	// can replace it with its newest result instead of starting a source copy.
	gate.observe(12U, started + std::chrono::milliseconds(500));
	CHECK_FALSE(take_if_input_quiet(false,
		started + std::chrono::milliseconds(999)).has_value());

	const auto latest = take_if_input_quiet(
		false, started + std::chrono::milliseconds(1000));
	REQUIRE(latest.has_value());
	CHECK(latest->identity.generation == 2U);
	CHECK_FALSE(mailbox.take().has_value());
}
