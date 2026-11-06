#ifndef __KVM_INTERFACE_LOGGER_HPP__
#define __KVM_INTERFACE_LOGGER_HPP__

#include <memory>
#include <string>

#include <spdlog/common.h>
#include <spdlog/spdlog.h>

class ILogger {
  public:
    virtual void logger_init(const std::string& level,
                             const std::string& type,
                             const std::string& ofilepath = "") = 0;
    virtual std::shared_ptr<spdlog::logger> operator->() = 0;
    virtual ~ILogger() = default;
};

class GlobalLogger {
  public:
    GlobalLogger() = delete;
    ~GlobalLogger() = delete;

    static void init(const std::string& level)
    {
        spdlog::set_level(spdlog::level::from_str(level));
    }
};

#endif  // __KVM_INTERFACE_LOGGER_HPP__
