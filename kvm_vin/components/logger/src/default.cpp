#include <logger/default.hpp>
#include <spdlog/sinks/basic_file_sink.h>
#include <spdlog/sinks/rotating_file_sink.h>
#include <spdlog/sinks/stdout_color_sinks.h>

constexpr int LOG_FILE_NUMBER = 3;
constexpr int LOG_FILE_SIZE = 2;

DefaultLogger::DefaultLogger(const std::string& name)
    : _name {std::move(name)}
{}

void DefaultLogger::logger_init(const std::string& level,
                                const std::string& type,
                                const std::string& ofilepath)
{
    std::vector<spdlog::sink_ptr> sinks;
    if (type == "console") {
        auto console_sink =
            std::make_shared<spdlog::sinks::stdout_color_sink_mt>();
        console_sink->set_color(spdlog::level::trace, console_sink->cyan);
        console_sink->set_color(spdlog::level::debug, console_sink->blue);
        console_sink->set_color(spdlog::level::info, console_sink->green);
        console_sink->set_color(spdlog::level::warn, console_sink->yellow);
        console_sink->set_color(spdlog::level::err, console_sink->red);
        console_sink->set_color(spdlog::level::critical, console_sink->red);
        sinks.push_back(console_sink);
    } else if (type == "file" && !ofilepath.empty()) {
        // sinks.push_back(
        //     std::make_shared<spdlog::sinks::basic_file_sink_mt>(ofilepath));
        auto rotating_sink =
            std::make_shared<spdlog::sinks::rotating_file_sink_mt>(
                ofilepath, 1024 * 1024 * LOG_FILE_SIZE, LOG_FILE_NUMBER);
        sinks.push_back(rotating_sink);
    } else {
        throw std::runtime_error("Unsupported sink type or missing file path: "
                                 + type);
    }

    this->_logger =
        std::make_shared<spdlog::logger>(this->_name, begin(sinks), end(sinks));
    const auto ilevel = spdlog::level::from_str(level);
    this->_logger->set_level(std::max(ilevel, spdlog::get_level()));
    this->_logger->set_pattern("[%Y-%m-%d %H:%M:%S.%e] [%n] [%^%l%$] %v");
}

std::shared_ptr<spdlog::logger> DefaultLogger::operator->()
{
    return this->_logger;
}
