#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/ssid_raster_worker.hpp"

#include <algorithm>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <limits>
#include <memory>
#include <mutex>
#include <optional>
#include <sstream>
#include <stdexcept>
#include <thread>
#include <vector>

using namespace std::chrono_literals;
using namespace lvgl_main;

namespace
{

std::filesystem::path test_font_path()
{
	return std::filesystem::exists("/usr/share/fonts/NotoSans-Regular.ttf")
		       ? std::filesystem::path("/usr/share/fonts/NotoSans-Regular.ttf")
		       : std::filesystem::path("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf");
}

#if defined(__linux__)
std::size_t font_mapping_count(const std::filesystem::path &font_path)
{
	std::ifstream maps("/proc/self/maps");
	REQUIRE(maps.good());
	const std::string suffix = " " + std::filesystem::canonical(font_path).string();
	std::size_t count = 0;
	for (std::string line; std::getline(maps, line);) {
		if (!line.ends_with(suffix)) {
			continue;
		}
		std::istringstream fields(line);
		std::string address, permissions;
		fields >> address >> permissions;
		CHECK(permissions == "r--p");
		++count;
	}
	return count;
}
#endif

ui::SsidRasterRequest request_for(std::uint64_t generation)
{
	ui::SsidRasterRequest request;
	request.identity.page_token = 41;
	request.identity.generation = generation;
	request.identity.font_generation = 1;
	request.identity.viewport_width = 150;
	request.identity.height = 32;
	request.identity.font_size_px = 20;
	request.identity.byte_budget = 4U * 1024U * 1024U;
	request.identity.display_text = "redacted";
	return request;
}

ui::SsidPreparedRasterResult completed_result(const ui::SsidRasterRequest &request)
{
	auto pixels = std::make_shared<const std::vector<std::uint32_t>>(
	    static_cast<std::size_t>(request.identity.viewport_width) * request.identity.height,
	    0xffffffffU);
	auto raster = std::make_shared<ui::SsidPreparedRaster>();
	raster->identity = request.identity;
	raster->source_width = request.identity.viewport_width;
	raster->source_height = request.identity.height;
	raster->stride_bytes =
	    static_cast<std::uint32_t>(request.identity.viewport_width) * sizeof(std::uint32_t);
	raster->text_width = request.identity.viewport_width / 2;
	raster->space_advance = 1;
	raster->pixels = std::move(pixels);
	REQUIRE(raster->has_valid_storage());
	return {.identity = request.identity, .raster = std::move(raster)};
}

} // namespace

TEST_CASE("SSID raster worker cancels an in-flight request and keeps only C after A B C")
{
	struct State {
		std::mutex mutex;
		std::condition_variable wake;
		bool first_started {false};
		bool release_first {false};
		std::vector<std::uint64_t> completions;
	} state;

	SsidRasterWorkerConfig config;
	config.font_size_px = 20;
	config.max_text_bytes = 512;

	SsidRasterWorker worker(
	    config,
	    [&state](ui::SsidPreparedRasterResult result) {
		    const std::lock_guard lock(state.mutex);
		    state.completions.push_back(result.identity.generation);
		    state.wake.notify_all();
	    },
	    [&state](const ui::SsidRasterRequest &request,
		     const SsidRasterWorker::CancelPredicate &cancelled) {
		    if (request.identity.generation == 1) {
			    std::unique_lock lock(state.mutex);
			    state.first_started = true;
			    state.wake.notify_all();
			    state.wake.wait(lock, [&state] { return state.release_first; });
		    }
		    if (cancelled()) {
			    return ui::SsidPreparedRasterResult {.identity = request.identity};
		    }
		    return completed_result(request);
	    });

	REQUIRE(worker.submit(request_for(1)));
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(state.wake.wait_for(lock, 2s, [&state] { return state.first_started; }));
	}
	REQUIRE(worker.submit(request_for(2)));
	REQUIRE(worker.submit(request_for(3)));
	{
		const std::lock_guard lock(state.mutex);
		state.release_first = true;
		state.wake.notify_all();
	}
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(state.wake.wait_for(lock, 2s,
					    [&state] { return state.completions.size() == 1; }));
		REQUIRE(state.completions.size() == 1);
		CHECK(state.completions.front() == 3);
	}
	worker.close();
}

TEST_CASE("SSID raster worker rejects requests after close")
{
	SsidRasterWorkerConfig config;
	config.font_size_px = 20;
	SsidRasterWorker worker(config, [](ui::SsidPreparedRasterResult) {});

	worker.close();
	CHECK_FALSE(worker.submit(request_for(1)));
}

TEST_CASE("SSID raster worker renders real font bytes without LVGL")
{
	const std::filesystem::path font_path = test_font_path();
	if (!std::filesystem::exists(font_path)) {
		WARN("No host TrueType font is available for the worker smoke test");
		return;
	}
#if defined(__linux__)
	const auto initial_mappings = font_mapping_count(font_path);
#endif

	struct State {
		std::mutex mutex;
		std::condition_variable wake;
		std::optional<ui::SsidPreparedRasterResult> result;
	} state;

	SsidRasterWorkerConfig config;
	config.base_font_path = font_path.string();
	config.cjk_font_path = font_path.string();
	config.font_size_px = 20;
	SsidRasterWorker worker(config, [&state](ui::SsidPreparedRasterResult result) {
		const std::lock_guard lock(state.mutex);
		state.result = std::move(result);
		state.wake.notify_all();
	});

	ui::SsidRasterRequest request = request_for(1);
	request.identity.display_text = "Office WiFi";
	REQUIRE(worker.submit(request));
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(
		    state.wake.wait_for(lock, 5s, [&state] { return state.result.has_value(); }));
		REQUIRE(state.result->is_success());
		CHECK(state.result->raster->source_width == request.identity.viewport_width);
		CHECK(state.result->raster->source_height == request.identity.height);
		CHECK(state.result->raster->text_width > 0);
		CHECK(std::any_of(state.result->raster->pixels->begin(),
				  state.result->raster->pixels->end(),
				  [](uint32_t pixel) { return (pixel >> 24U) != 0U; }));
	}
#if defined(__linux__)
	CHECK(font_mapping_count(font_path) > initial_mappings);
#endif

	request = request_for(2);
	request.identity.scroll_enabled = true;
	request.identity.display_text.assign(80, 'W');
	{
		const std::lock_guard lock(state.mutex);
		state.result.reset();
	}
	REQUIRE(worker.submit(request));
	uint16_t source_width = 0;
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(state.wake.wait_for(lock, 5s, [&state] {
			return state.result.has_value() &&
			       state.result->identity.generation == 2;
		}));
		REQUIRE(state.result->is_success());
		REQUIRE(state.result->raster->overflows_viewport);
		source_width = state.result->raster->source_width;
	}

	request.identity.generation = 3;
	const std::size_t source_bytes =
		static_cast<std::size_t>(source_width) * request.identity.height *
		sizeof(uint32_t);
	const std::size_t window_bytes =
		static_cast<std::size_t>(request.identity.viewport_width) *
		request.identity.height * sizeof(uint32_t);
	REQUIRE(source_bytes + window_bytes - 1U <=
		std::numeric_limits<uint32_t>::max());
	request.identity.byte_budget =
		static_cast<uint32_t>(source_bytes + window_bytes - 1U);
	{
		const std::lock_guard lock(state.mutex);
		state.result.reset();
	}
	REQUIRE(worker.submit(request));
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(state.wake.wait_for(lock, 5s, [&state] {
			return state.result.has_value() &&
			       state.result->identity.generation == 3;
		}));
		CHECK_FALSE(state.result->is_success());
	}
	worker.close();
#if defined(__linux__)
	CHECK(font_mapping_count(font_path) == initial_mappings);
#endif
}

TEST_CASE("SSID raster worker releases font mappings when loading fails")
{
	const std::filesystem::path font_path = test_font_path();
	if (!std::filesystem::exists(font_path)) {
		WARN("No host TrueType font is available for the worker failure test");
		return;
	}

	struct TempDir {
		std::filesystem::path path = std::filesystem::temp_directory_path() /
			("nanokvm_ssid_font_" + std::to_string(
				std::chrono::steady_clock::now().time_since_epoch().count()));
		TempDir() { std::filesystem::create_directory(path); }
		~TempDir()
		{
			std::error_code error;
			std::filesystem::remove_all(path, error);
		}
	} temp;
	const auto invalid_font = temp.path / "invalid.ttf";
	{
		std::ofstream file(invalid_font, std::ios::binary);
		REQUIRE(file.good());
	}

	SsidRasterWorkerConfig config;
	config.base_font_path = "A:" + font_path.string();
	config.cjk_font_path = invalid_font.string();
	SECTION("empty font") {}
	SECTION("invalid font header")
	{
		std::ofstream file(invalid_font, std::ios::binary);
		file << "not a TrueType or OpenType font";
		REQUIRE(file.good());
	}
	SECTION("missing font") { config.cjk_font_path = (temp.path / "missing.ttf").string(); }
	SECTION("directory") { config.cjk_font_path = temp.path.string(); }
	SECTION("relative path") { config.cjk_font_path = "invalid.ttf"; }
	SECTION("font exceeds configured limit") { config.max_font_bytes = 1; }
	SECTION("zero limit") { config.max_font_bytes = 0; }
	SECTION("limit exceeds hard maximum") { config.max_font_bytes = 64U * 1024U * 1024U + 1U; }

#if defined(__linux__)
	const auto initial_mappings = font_mapping_count(font_path);
	const auto initial_invalid_mappings = font_mapping_count(invalid_font);
#endif
	std::mutex mutex;
	std::condition_variable wake;
	std::optional<ui::SsidPreparedRasterResult> result;
	SsidRasterWorker worker(config, [&](ui::SsidPreparedRasterResult completed) {
		const std::lock_guard lock(mutex);
		result = std::move(completed);
		wake.notify_all();
	});
	REQUIRE(worker.submit(request_for(1)));
	{
		std::unique_lock lock(mutex);
		REQUIRE(wake.wait_for(lock, 5s, [&] { return result.has_value(); }));
		CHECK_FALSE(result->is_success());
	}
#if defined(__linux__)
	CHECK(font_mapping_count(font_path) == initial_mappings);
	CHECK(font_mapping_count(invalid_font) == initial_invalid_mappings);
#endif
	worker.close();
}

TEST_CASE("SSID raster worker converts rasterizer exceptions to failure results")
{
	struct State {
		std::mutex mutex;
		std::condition_variable wake;
		std::optional<ui::SsidPreparedRasterResult> result;
	} state;

	SsidRasterWorkerConfig config;
	config.font_size_px = 20;
	SsidRasterWorker worker(
	    config,
	    [&state](ui::SsidPreparedRasterResult result) {
		    const std::lock_guard lock(state.mutex);
		    state.result = std::move(result);
		    state.wake.notify_all();
	    },
	    [](const ui::SsidRasterRequest &,
	       const SsidRasterWorker::CancelPredicate &) -> ui::SsidPreparedRasterResult {
		    throw std::runtime_error("injected raster failure");
	    });

	REQUIRE(worker.submit(request_for(1)));
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(
		    state.wake.wait_for(lock, 2s, [&state] { return state.result.has_value(); }));
		CHECK(state.result->identity == request_for(1).identity);
		CHECK_FALSE(state.result->is_success());
	}
	worker.close();
}

TEST_CASE("SSID raster worker survives a completion callback exception")
{
	struct State {
		std::mutex mutex;
		std::condition_variable wake;
		std::size_t callbacks = 0;
	} state;

	SsidRasterWorkerConfig config;
	config.font_size_px = 20;
	SsidRasterWorker worker(
	    config,
	    [&state](ui::SsidPreparedRasterResult) {
		    std::size_t callbacks = 0;
		    {
			    const std::lock_guard lock(state.mutex);
			    callbacks = ++state.callbacks;
			    state.wake.notify_all();
		    }
		    if (callbacks == 1U) {
			    throw std::runtime_error("injected completion failure");
		    }
	    },
	    [](const ui::SsidRasterRequest &request, const SsidRasterWorker::CancelPredicate &) {
		    return completed_result(request);
	    });

	REQUIRE(worker.submit(request_for(1)));
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(state.wake.wait_for(lock, 2s, [&state] { return state.callbacks == 1U; }));
	}
	REQUIRE(worker.submit(request_for(2)));
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(state.wake.wait_for(lock, 2s, [&state] { return state.callbacks == 2U; }));
	}
	worker.close();
}

TEST_CASE("SSID raster worker cancels in-flight work before close returns")
{
	struct State {
		std::mutex mutex;
		std::condition_variable wake;
		bool started {false};
		bool cancellation_seen {false};
		std::size_t completions = 0;
	} state;

	SsidRasterWorkerConfig config;
	config.font_size_px = 20;
	SsidRasterWorker worker(
	    config,
	    [&state](ui::SsidPreparedRasterResult) {
		    const std::lock_guard lock(state.mutex);
		    ++state.completions;
	    },
	    [&state](const ui::SsidRasterRequest &request,
		     const SsidRasterWorker::CancelPredicate &cancelled) {
		    {
			    const std::lock_guard lock(state.mutex);
			    state.started = true;
			    state.wake.notify_all();
		    }
		    while (!cancelled()) {
			    std::this_thread::sleep_for(1ms);
		    }
		    {
			    const std::lock_guard lock(state.mutex);
			    state.cancellation_seen = true;
		    }
		    return ui::SsidPreparedRasterResult {
			.identity = request.identity,
		    };
	    });

	REQUIRE(worker.submit(request_for(1)));
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(state.wake.wait_for(lock, 2s, [&state] { return state.started; }));
	}
	worker.close();
	CHECK(state.cancellation_seen);
	CHECK(state.completions == 0U);
}

TEST_CASE("SSID raster worker invalidation suppresses stale completion and permits retry")
{
	struct State {
		std::mutex mutex;
		std::condition_variable wake;
		bool first_started {false};
		bool release_first {false};
		std::vector<std::uint64_t> completions;
	} state;

	SsidRasterWorkerConfig config;
	config.font_size_px = 20;
	SsidRasterWorker worker(
	    config,
	    [&state](ui::SsidPreparedRasterResult result) {
		    const std::lock_guard lock(state.mutex);
		    state.completions.push_back(result.identity.generation);
		    state.wake.notify_all();
	    },
	    [&state](const ui::SsidRasterRequest &request,
		     const SsidRasterWorker::CancelPredicate &cancelled) {
		    if (request.identity.generation == 1) {
			    std::unique_lock lock(state.mutex);
			    state.first_started = true;
			    state.wake.notify_all();
			    state.wake.wait(lock, [&state] { return state.release_first; });
		    }
		    if (cancelled()) {
			    return ui::SsidPreparedRasterResult {
				.identity = request.identity,
			    };
		    }
		    return completed_result(request);
	    });

	REQUIRE(worker.submit(request_for(1)));
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(state.wake.wait_for(lock, 2s, [&state] { return state.first_started; }));
	}
	worker.invalidate_page(41);
	{
		const std::lock_guard lock(state.mutex);
		state.release_first = true;
		state.wake.notify_all();
	}
	REQUIRE(worker.submit(request_for(2)));
	{
		std::unique_lock lock(state.mutex);
		REQUIRE(state.wake.wait_for(lock, 2s,
					    [&state] { return state.completions.size() == 1U; }));
		CHECK(state.completions == std::vector<std::uint64_t> {2});
	}
	worker.close();
}

TEST_CASE("SSID prepared raster rejects inconsistent storage geometry")
{
	const ui::SsidRasterRequest request = request_for(1);
	const ui::SsidPreparedRasterResult result = completed_result(request);
	REQUIRE(result.is_success());

	ui::SsidPreparedRaster malformed = *result.raster;
	malformed.stride_bytes -= 1U;
	CHECK_FALSE(malformed.has_valid_storage());

	malformed = *result.raster;
	malformed.source_width += 1U;
	CHECK_FALSE(malformed.has_valid_storage());

	malformed = *result.raster;
	malformed.identity.byte_budget = 1U;
	CHECK_FALSE(malformed.has_valid_storage());

	malformed = *result.raster;
	malformed.overflows_viewport = true;
	CHECK_FALSE(malformed.has_valid_storage());

	malformed = *result.raster;
	malformed.text_width = malformed.identity.viewport_width + 1;
	CHECK(malformed.has_valid_storage());
	malformed.identity.scroll_enabled = true;
	CHECK_FALSE(malformed.has_valid_storage());
}

TEST_CASE("Overflowing SSID raster budget includes its fixed visible window")
{
	ui::SsidRasterRequest request = request_for(1);
	request.identity.scroll_enabled = true;
	constexpr uint16_t source_width = 200;
	const std::size_t source_bytes =
		static_cast<std::size_t>(source_width) * request.identity.height *
		sizeof(uint32_t);
	const std::size_t window_bytes =
		static_cast<std::size_t>(request.identity.viewport_width) *
		request.identity.height * sizeof(uint32_t);
	REQUIRE(source_bytes + window_bytes <=
		std::numeric_limits<uint32_t>::max());
	request.identity.byte_budget =
		static_cast<uint32_t>(source_bytes + window_bytes);

	ui::SsidPreparedRaster raster;
	raster.identity = request.identity;
	raster.source_width = source_width;
	raster.source_height = request.identity.height;
	raster.stride_bytes = source_width * sizeof(uint32_t);
	raster.text_width = source_width;
	raster.space_advance = 5;
	raster.overflows_viewport = true;
	raster.pixels = std::make_shared<const std::vector<uint32_t>>(
		static_cast<std::size_t>(source_width) * request.identity.height,
		0xffffffffU);
	REQUIRE(raster.has_valid_storage());

	raster.identity.byte_budget -= 1U;
	CHECK_FALSE(raster.has_valid_storage());
}
