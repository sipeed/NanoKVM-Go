#include <fmt/format.h>
#include <tools/tools.hpp>

namespace tools {

fmt::memory_buffer& thread_buffer()
{
    thread_local fmt::memory_buffer buf;
    buf.clear();
    return buf;
}

std::string_view tag(bool file_line, bool func, SourceLocation loc)
{
    auto& buf = thread_buffer();
    if (file_line && func) {
        fmt::format_to(std::back_inserter(buf),
                       "[{}({}) {}]",
                       loc.file_name(),
                       loc.line(),
                       loc.function_name());
    } else if (!file_line && func) {
        fmt::format_to(std::back_inserter(buf), "[{}]", loc.function_name());
    } else if (file_line && !func) {
        fmt::format_to(
            std::back_inserter(buf), "[{}({})]", loc.file_name(), loc.line());
    } else {
        fmt::format_to(std::back_inserter(buf), "");
    }
    return std::string_view(buf.data(), buf.size());
}

}  // namespace tools
