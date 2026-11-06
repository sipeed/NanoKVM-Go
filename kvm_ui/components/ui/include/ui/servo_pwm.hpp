#pragma once

#include <cstdint>
#include <filesystem>
#include <memory>
#include <optional>
#include <string>
#include <vector>

namespace ui::servo_pwm
{

inline constexpr int kPeriodNs = 20000000;
inline constexpr int kDefaultIdleNs = 1500000;
inline constexpr int kIdleMinNs = 100000;
inline constexpr int kIdleMaxNs = 3500000;
inline constexpr uint32_t kClickStepDelayMs = 120;

struct Config {
	std::filesystem::path pwm_chip { "/sys/class/pwm/pwmchip0" };
	std::vector<int> channels { 0, 1 };
	int period_ns { kPeriodNs };
	std::vector<uint32_t> mux_addrs { 0x02304084U, 0x02304090U };
	uint32_t mux_value { 0x00030003U };
};

struct Step {
	int duty_ns { 0 };
	uint32_t delay_after_ms { 0 };
};

struct ApplyResult {
	bool ok { false };
	std::string error;
};

class Backend {
    public:
	virtual ~Backend() = default;

	virtual ApplyResult initialize(int idle_ns) = 0;
	virtual ApplyResult write_idle(int idle_ns) = 0;
};

class ProductionBackend final : public Backend {
    public:
	explicit ProductionBackend(Config config = {});

	ApplyResult initialize(int idle_ns) override;
	ApplyResult write_idle(int idle_ns) override;

    private:
	Config _config;
};

struct InMemoryBackendOptions {
	bool fail_initialize { false };
	bool fail_writes { false };
};

class InMemoryBackend final : public Backend {
    public:
	explicit InMemoryBackend(InMemoryBackendOptions options = {});

	ApplyResult initialize(int idle_ns) override;
	ApplyResult write_idle(int idle_ns) override;

	bool is_initialized() const;
	std::optional<int> current_duty_ns() const;
	const std::vector<int> &write_history() const;

    private:
	InMemoryBackendOptions _options;
	bool _initialized { false };
	std::optional<int> _current_duty_ns;
	std::vector<int> _write_history;
};

int clamp_idle_ns(int idle_ns);
std::string format_idle_ms(int idle_ns);
std::string format_idle_compact_ms(int idle_ns);
std::vector<Step> build_free_mode_plan(int idle_ns);
std::vector<Step> build_click_mode_plan(int idle_ns, int zero_ns);

ApplyResult initialize(const Config &config, int idle_ns);
ApplyResult write_idle(const Config &config, int idle_ns);
ApplyResult apply_plan(const Config &config, const std::vector<Step> &plan,
			       bool initialize_pwm = true);
std::unique_ptr<Backend> make_production_backend(Config config = {});
std::unique_ptr<Backend> make_default_backend();

} // namespace ui::servo_pwm
