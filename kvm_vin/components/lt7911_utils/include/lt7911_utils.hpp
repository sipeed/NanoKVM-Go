#pragma once

#include <logger/ilogger.hpp>

class Lt7911Utils final : public ILogger {
  public:
    Lt7911Utils(const std::string& proc_base_path = "/proc/lt7911_info");
    ~Lt7911Utils() = default;

    bool read_status(std::string& status);
    bool write_status(const std::string& status);

    bool read_res(int& width, int& height, int& fps);

    bool reset_lt7911(bool enable);

    /* Logger */
    void logger_init(const std::string& level,
                     const std::string& type,
                     const std::string& ofilepath = "") override;
    std::shared_ptr<spdlog::logger> operator->() override;

  private:
    std::shared_ptr<ILogger> _logger {nullptr};
    std::string _base_path;
    bool _read_int_from_file(const std::string& filename, int& value);
};
