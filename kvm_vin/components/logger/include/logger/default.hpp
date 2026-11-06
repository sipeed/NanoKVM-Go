#ifndef __KVM_LOGGER_DEFAULT_HPP__
#define __KVM_LOGGER_DEFAULT_HPP__

#include "ilogger.hpp"

class DefaultLogger final : public ILogger {
  public:
    explicit DefaultLogger(const std::string& name);
    virtual void logger_init(const std::string& level,
                             const std::string& type,
                             const std::string& ofilepath = "") override;
    virtual std::shared_ptr<spdlog::logger> operator->() override;

  private:
    std::string _name;
    std::shared_ptr<spdlog::logger> _logger;
};

#endif  // __KVM_LOGGER_DEFAULT_HPP__
