#include "ui/servo_pwm.hpp"

#include <algorithm>
#include <array>
#include <cerrno>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <memory>
#include <sstream>
#include <string_view>
#include <thread>
#include <utility>

#include <spdlog/spdlog.h>

namespace ui::servo_pwm
{

namespace
{

bool write_text_file(const std::filesystem::path &path, std::string_view text,
		     std::string &error)
{
	std::FILE *file = std::fopen(path.c_str(), "we");
	if (file == nullptr) {
		error = path.string() + ": " + std::strerror(errno);
		return false;
	}
	std::unique_ptr<std::FILE, decltype(&std::fclose)> close_file(
		file, &std::fclose);

	const auto written = std::fwrite(text.data(), 1U, text.size(), file);
	if (written != text.size() || std::fflush(file) != 0) {
		error = path.string() + ": write failed";
		return false;
	}
	return true;
}

std::string hex32(uint32_t value)
{
	std::array<char, 11> buffer {};
	std::snprintf(buffer.data(), buffer.size(), "0x%08x", value);
	return buffer.data();
}

bool configure_mux(const Config &config, std::string &error)
{
	for (const uint32_t addr : config.mux_addrs) {
		const std::string command = "devmem " + hex32(addr) + " 32 " +
					    hex32(config.mux_value) +
					    " >/dev/null";
		const int rc = std::system(command.c_str());
		if (rc != 0) {
			error = "devmem failed for " + hex32(addr);
			return false;
		}
	}
	return true;
}

bool wait_for_pwm(const std::filesystem::path &pwm_path, std::string &error)
{
	for (int attempt = 0; attempt < 20; ++attempt) {
		std::error_code ec;
		if (std::filesystem::is_directory(pwm_path, ec)) {
			return true;
		}
		std::this_thread::sleep_for(std::chrono::milliseconds(25));
	}
	error = "pwm path was not created: " + pwm_path.string();
	return false;
}

bool init_channel(const Config &config, int channel, int idle_ns,
		  std::string &error)
{
	std::error_code ec;
	if (!std::filesystem::is_directory(config.pwm_chip, ec)) {
		error = "pwm chip not found: " + config.pwm_chip.string();
		return false;
	}

	const std::filesystem::path pwm_path =
		config.pwm_chip / ("pwm" + std::to_string(channel));
	if (!std::filesystem::is_directory(pwm_path, ec)) {
		if (!write_text_file(config.pwm_chip / "export",
				     std::to_string(channel), error)) {
			return false;
		}
		if (!wait_for_pwm(pwm_path, error)) {
			return false;
		}
	}

	std::string ignored_error;
	(void)write_text_file(pwm_path / "enable", "0", ignored_error);
	if (!write_text_file(pwm_path / "period",
			     std::to_string(config.period_ns), error)) {
		return false;
	}
	if (!write_text_file(pwm_path / "duty_cycle", std::to_string(idle_ns),
			     error)) {
		return false;
	}
	return write_text_file(pwm_path / "enable", "1", error);
}

bool write_channel_idle(const Config &config, int channel, int idle_ns,
			std::string &error)
{
	const std::filesystem::path pwm_path =
		config.pwm_chip / ("pwm" + std::to_string(channel));
	std::error_code ec;
	if (!std::filesystem::is_directory(pwm_path, ec)) {
		return init_channel(config, channel, idle_ns, error);
	}
	return write_text_file(pwm_path / "duty_cycle", std::to_string(idle_ns),
			       error);
}

ApplyResult ok_result()
{
	ApplyResult result;
	result.ok = true;
	return result;
}

ApplyResult error_result(std::string error)
{
	ApplyResult result;
	result.ok = false;
	result.error = std::move(error);
	return result;
}

} // namespace

int clamp_idle_ns(int idle_ns)
{
	return std::clamp(idle_ns, kIdleMinNs, kIdleMaxNs);
}

std::string format_idle_ms(int idle_ns)
{
	std::array<char, 16> buffer {};
	std::snprintf(buffer.data(), buffer.size(), "%.3fms",
		      static_cast<double>(clamp_idle_ns(idle_ns)) / 1000000.0);
	return buffer.data();
}

std::string format_idle_compact_ms(int idle_ns)
{
	std::array<char, 16> buffer {};
	std::snprintf(buffer.data(), buffer.size(), "%.2f",
		      static_cast<double>(clamp_idle_ns(idle_ns)) / 1000000.0);
	return buffer.data();
}

std::vector<Step> build_free_mode_plan(int idle_ns)
{
	return { Step { .duty_ns = clamp_idle_ns(idle_ns),
			.delay_after_ms = 0 } };
}

std::vector<Step> build_click_mode_plan(int idle_ns, int zero_ns)
{
	const int zero_duty_ns = clamp_idle_ns(zero_ns);
	return { Step { .duty_ns = zero_duty_ns,
			.delay_after_ms = kClickStepDelayMs },
		 Step { .duty_ns = clamp_idle_ns(idle_ns),
			.delay_after_ms = kClickStepDelayMs },
		 Step { .duty_ns = zero_duty_ns, .delay_after_ms = 0 } };
}

ApplyResult initialize(const Config &config, int idle_ns)
{
	std::string error;
	if (!configure_mux(config, error)) {
		return error_result(std::move(error));
	}

	const int duty_ns = clamp_idle_ns(idle_ns);
	for (const int channel : config.channels) {
		if (!init_channel(config, channel, duty_ns, error)) {
			return error_result(std::move(error));
		}
	}
	return ok_result();
}

ApplyResult write_idle(const Config &config, int idle_ns)
{
	std::string error;
	const int duty_ns = clamp_idle_ns(idle_ns);
	for (const int channel : config.channels) {
		if (!write_channel_idle(config, channel, duty_ns, error)) {
			return error_result(std::move(error));
		}
	}
	return ok_result();
}

ProductionBackend::ProductionBackend(Config config)
	: _config(std::move(config))
{
}

ApplyResult ProductionBackend::initialize(int idle_ns)
{
	return ::ui::servo_pwm::initialize(_config, idle_ns);
}

ApplyResult ProductionBackend::write_idle(int idle_ns)
{
	return ::ui::servo_pwm::write_idle(_config, idle_ns);
}

std::unique_ptr<Backend> make_production_backend(Config config)
{
	return std::make_unique<ProductionBackend>(std::move(config));
}

InMemoryBackend::InMemoryBackend(InMemoryBackendOptions options)
	: _options(options)
{
}

ApplyResult InMemoryBackend::initialize(int idle_ns)
{
	if (_options.fail_initialize) {
		return error_result(
			"in-memory PWM initialization failure injected");
	}

	const int duty_ns = clamp_idle_ns(idle_ns);
	_initialized = true;
	_current_duty_ns = duty_ns;
	_write_history.push_back(duty_ns);
	return ok_result();
}

ApplyResult InMemoryBackend::write_idle(int idle_ns)
{
	if (_options.fail_writes) {
		return error_result("in-memory PWM write failure injected");
	}

	const int duty_ns = clamp_idle_ns(idle_ns);
	_current_duty_ns = duty_ns;
	_write_history.push_back(duty_ns);
	return ok_result();
}

bool InMemoryBackend::is_initialized() const
{
	return _initialized;
}

std::optional<int> InMemoryBackend::current_duty_ns() const
{
	return _current_duty_ns;
}

const std::vector<int> &InMemoryBackend::write_history() const
{
	return _write_history;
}

std::unique_ptr<Backend> make_default_backend()
{
#if NANOKVM_GO_SIMULATOR_BUILD
	spdlog::info("[servo] Servo PWM uses the in-memory simulator backend");
	return std::make_unique<InMemoryBackend>();
#else
	return make_production_backend();
#endif
}

ApplyResult apply_plan(const Config &config, const std::vector<Step> &plan,
		       bool initialize_pwm)
{
	if (plan.empty()) {
		return ok_result();
	}

	if (initialize_pwm) {
		const ApplyResult init_result =
			initialize(config, plan.front().duty_ns);
		if (!init_result.ok) {
			return init_result;
		}
	}

	for (const Step &step : plan) {
		const ApplyResult write_result =
			write_idle(config, step.duty_ns);
		if (!write_result.ok) {
			return write_result;
		}
		if (step.delay_after_ms > 0U) {
			std::this_thread::sleep_for(
				std::chrono::milliseconds(step.delay_after_ms));
		}
	}
	return ok_result();
}

} // namespace ui::servo_pwm
