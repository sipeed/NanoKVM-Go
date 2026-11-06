#include <chrono>
#include <filesystem>
#include <fstream>
#include <string>

#include <catch2/catch_test_macros.hpp>

#include "ui/servo_pwm.hpp"

using namespace ui::servo_pwm;

namespace
{

class TempDir final {
    public:
	TempDir()
	{
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_servo_pwm_test_" + std::to_string(stamp));
		std::filesystem::create_directories(_path);
	}

	~TempDir()
	{
		std::error_code ec;
		std::filesystem::remove_all(_path, ec);
	}

	TempDir(const TempDir &) = delete;
	TempDir &operator=(const TempDir &) = delete;

	const std::filesystem::path &path() const
	{
		return _path;
	}

    private:
	std::filesystem::path _path;
};

void write_text(const std::filesystem::path &path, const std::string &text)
{
	std::ofstream output(path, std::ios::trunc);
	REQUIRE(output.good());
	output << text;
	REQUIRE(output.good());
}

std::string read_text(const std::filesystem::path &path)
{
	std::ifstream input(path);
	std::string text;
	std::getline(input, text);
	return text;
}

} // namespace

TEST_CASE("servo pwm clamps idle duty to the configured range")
{
	CHECK(kIdleMinNs == 100000);
	CHECK(kIdleMaxNs == 3500000);
	CHECK(kDefaultIdleNs == 1500000);
	CHECK(clamp_idle_ns(-1) == kIdleMinNs);
	CHECK(clamp_idle_ns(0) == kIdleMinNs);
	CHECK(clamp_idle_ns(kIdleMaxNs + 1) == kIdleMaxNs);
	CHECK(clamp_idle_ns(kDefaultIdleNs) == kDefaultIdleNs);
}

TEST_CASE("servo free mode writes the selected idle duty once")
{
	const auto plan = build_free_mode_plan(1600000);

	REQUIRE(plan.size() == 1);
	CHECK(plan[0].duty_ns == 1600000);
	CHECK(plan[0].delay_after_ms == 0);
}

TEST_CASE("servo free mode clamps values below range to the minimum duty")
{
	const auto plan = build_free_mode_plan(0);

	REQUIRE(plan.size() == 1);
	CHECK(plan[0].duty_ns == kIdleMinNs);
	CHECK(plan[0].delay_after_ms == 0);
}

TEST_CASE("servo idle duty is formatted as milliseconds for the UI title")
{
	CHECK(format_idle_ms(kIdleMinNs) == "0.100ms");
	CHECK(format_idle_ms(2000000) == "2.000ms");
	CHECK(format_idle_ms(kIdleMaxNs) == "3.500ms");
	CHECK(format_idle_ms(0) == "0.100ms");
}

TEST_CASE("servo idle duty is formatted compactly for the click button")
{
	CHECK(format_idle_compact_ms(kIdleMinNs) == "0.10");
	CHECK(format_idle_compact_ms(kDefaultIdleNs) == "1.50");
	CHECK(format_idle_compact_ms(kIdleMaxNs) == "3.50");
	CHECK(format_idle_compact_ms(0) == "0.10");
}

TEST_CASE(
	"servo click mode returns to configured zero, visits selected duty, and returns")
{
	const auto plan = build_click_mode_plan(1600000, 1200000);

	REQUIRE(plan.size() == 3);
	CHECK(plan[0].duty_ns == 1200000);
	CHECK(plan[0].delay_after_ms == kClickStepDelayMs);
	CHECK(plan[1].duty_ns == 1600000);
	CHECK(plan[1].delay_after_ms == kClickStepDelayMs);
	CHECK(plan[2].duty_ns == 1200000);
	CHECK(plan[2].delay_after_ms == 0);
}

TEST_CASE("servo click mode clamps configured zero and target independently")
{
	const auto plan = build_click_mode_plan(kIdleMaxNs + 1, 0);

	REQUIRE(plan.size() == 3);
	CHECK(plan[0].duty_ns == kIdleMinNs);
	CHECK(plan[1].duty_ns == kIdleMaxNs);
	CHECK(plan[2].duty_ns == kIdleMinNs);
	CHECK(plan[2].delay_after_ms == 0);
}

TEST_CASE("servo default config mirrors the shell helper defaults")
{
	const Config config;

	CHECK(config.pwm_chip == "/sys/class/pwm/pwmchip0");
	REQUIRE(config.channels.size() == 2);
	CHECK(config.channels[0] == 0);
	CHECK(config.channels[1] == 1);
	CHECK(config.period_ns == 20000000);
	CHECK(config.mux_addrs.size() == 2);
	CHECK(config.mux_addrs[0] == 0x02304084U);
	CHECK(config.mux_addrs[1] == 0x02304090U);
	CHECK(config.mux_value == 0x00030003U);
}

TEST_CASE(
	"production servo backend owns its config and delegates clamped duties")
{
	TempDir temp;
	const auto channel = temp.path() / "pwmchip0" / "pwm0";
	std::filesystem::create_directories(channel);
	write_text(channel / "enable", "0");
	write_text(channel / "period", "0");
	write_text(channel / "duty_cycle", "0");

	ProductionBackend backend(Config {
		.pwm_chip = channel.parent_path(),
		.channels = { 0 },
		.mux_addrs = {},
	});
	REQUIRE(backend.initialize(kIdleMaxNs + 1).ok);
	CHECK(read_text(channel / "period") == std::to_string(kPeriodNs));
	CHECK(read_text(channel / "duty_cycle") == std::to_string(kIdleMaxNs));

	REQUIRE(backend.write_idle(0).ok);
	CHECK(read_text(channel / "duty_cycle") == std::to_string(kIdleMinNs));
}

TEST_CASE("in-memory servo backend records clamped successful writes")
{
	InMemoryBackend backend;

	CHECK_FALSE(backend.is_initialized());
	CHECK_FALSE(backend.current_duty_ns().has_value());
	CHECK(backend.write_history().empty());

	REQUIRE(backend.initialize(kIdleMaxNs + 1).ok);
	CHECK(backend.is_initialized());
	REQUIRE(backend.current_duty_ns().has_value());
	CHECK(*backend.current_duty_ns() == kIdleMaxNs);

	REQUIRE(backend.write_idle(0).ok);
	REQUIRE(backend.write_history().size() == 2U);
	CHECK(backend.write_history()[0] == kIdleMaxNs);
	CHECK(backend.write_history()[1] == kIdleMinNs);
	REQUIRE(backend.current_duty_ns().has_value());
	CHECK(*backend.current_duty_ns() == kIdleMinNs);
}

TEST_CASE("in-memory servo backend failure injection preserves state")
{
	InMemoryBackend initialize_failure(
		InMemoryBackendOptions { .fail_initialize = true });
	const ApplyResult initialize_result =
		initialize_failure.initialize(kDefaultIdleNs);
	CHECK_FALSE(initialize_result.ok);
	CHECK_FALSE(initialize_failure.is_initialized());
	CHECK_FALSE(initialize_failure.current_duty_ns().has_value());
	CHECK(initialize_failure.write_history().empty());

	InMemoryBackend write_failure(
		InMemoryBackendOptions { .fail_writes = true });
	REQUIRE(write_failure.initialize(kDefaultIdleNs).ok);
	const ApplyResult write_result = write_failure.write_idle(kIdleMaxNs);
	CHECK_FALSE(write_result.ok);
	REQUIRE(write_failure.current_duty_ns().has_value());
	CHECK(*write_failure.current_duty_ns() == kDefaultIdleNs);
	REQUIRE(write_failure.write_history().size() == 1U);
	CHECK(write_failure.write_history()[0] == kDefaultIdleNs);
}

TEST_CASE("servo default backend matches the selected build mode")
{
	const auto backend = make_default_backend();
	REQUIRE(backend != nullptr);

#if NANOKVM_GO_SIMULATOR_BUILD
	CHECK(dynamic_cast<InMemoryBackend *>(backend.get()) != nullptr);
#else
	CHECK(dynamic_cast<ProductionBackend *>(backend.get()) != nullptr);
#endif
}
