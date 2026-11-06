#ifndef __KVM_UI_TOOLS_SYSTEM_UPTIME_HPP__
#define __KVM_UI_TOOLS_SYSTEM_UPTIME_HPP__

#include <chrono>
#include <ctime>
#include <optional>
#include <system_error>

namespace tools {

class SystemUptime final {
  public:
    using DefaultDuration = std::chrono::seconds;

    template<typename T>
    using RT = std::optional<T>;

    SystemUptime() = delete;
    ~SystemUptime() = default;

    SystemUptime(const SystemUptime&) = delete;
    SystemUptime& operator=(const SystemUptime&) = delete;
    SystemUptime(SystemUptime&&) = delete;
    SystemUptime& operator=(SystemUptime&&) = delete;

    template<typename Duration = DefaultDuration>
    static RT<Duration> now() noexcept
    {
        timespec ts;
        if (::clock_gettime(CLOCK_REALTIME, &ts) != 0) {
            return std::nullopt;
        }

        auto total_ns = std::chrono::seconds(ts.tv_sec)
            + std::chrono::nanoseconds(ts.tv_nsec);
        return std::chrono::floor<Duration>(total_ns);
    }

    template<typename Duration = DefaultDuration>
    static Duration now2() noexcept
    {
        while (true) {
            auto result = now<Duration>();
            if (result.has_value()) {
                return result.value();
            }
        }
    }

    template<typename Duration>
    static Duration elapsed(const Duration& prev) noexcept
    {
        const auto now = now2<Duration>();
        return now - prev;
    }
};

}  // namespace tools

#endif  // __KVM_UI_TOOLS_SYSTEM_UPTIME_HPP__
