#ifndef __KVM_UI_LOGGERS_TOOLS_HPP__
#define __KVM_UI_LOGGERS_TOOLS_HPP__

#include <string>

#include <spdlog/spdlog.h>

class LoggerTools final {
  public:
    LoggerTools() = delete;
    ~LoggerTools() = default;

    LoggerTools(const LoggerTools&) = delete;
    LoggerTools& operator=(const LoggerTools&) = delete;
    LoggerTools(LoggerTools&&) = delete;
    LoggerTools& operator=(LoggerTools&&) = delete;

  public:
    static std::string adjust_log_level(const std::string& level);
    static std::string adjust_log_level(const std::string& glevel,
                                        const std::string& ilevel);
};

#endif  // __KVM_UI_LOGGERS_TOOLS_HPP__
