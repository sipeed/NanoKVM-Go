#include <logger/tools.hpp>
#include <spdlog/common.h>

std::string LoggerTools::adjust_log_level(const std::string& level)
{
    using namespace spdlog::level;
    return {to_string_view(from_str(level)).data()};
}

std::string LoggerTools::adjust_log_level(const std::string& glevel,
                                          const std::string& ilevel)
{
    using namespace spdlog::level;
    auto global_level = from_str(glevel);
    auto input_level = from_str(ilevel);

    if (global_level == off) {
        return {to_string_view(off).data()};
    }

    if (input_level == off) {
        return {to_string_view(global_level).data()};
    }

    if (static_cast<int>(input_level) > static_cast<int>(global_level)) {
        input_level = global_level;
    }

    return {to_string_view(input_level).data()};
}
