#ifndef __KVM_UI_TOOLS_HPP__
#define __KVM_UI_TOOLS_HPP__

#include <cstdint>
#include <shared_mutex>
#include <string_view>

namespace tools {

struct SourceLocation {
    static constexpr SourceLocation current(
        const char* file_name = __builtin_FILE(),
        const char* function_name = __builtin_FUNCTION(),
        uint_least32_t line = __builtin_LINE()) noexcept
    {
        SourceLocation loc;
        loc.file_name_ = file_name;
        loc.function_name_ = function_name;
        loc.line_ = line;
        return loc;
    }

    constexpr SourceLocation() noexcept = default;

    constexpr const char* file_name() const noexcept { return file_name_; }
    constexpr const char* function_name() const noexcept { return function_name_; }
    constexpr uint_least32_t line() const noexcept { return line_; }

  private:
    const char* file_name_ = "";
    const char* function_name_ = "";
    uint_least32_t line_ = 0;
};

template<typename T>
struct Guarded {
    std::shared_mutex mutex;
    T data;
};

std::string_view tag(
    bool file_line = false,
    bool func = true,
    SourceLocation loc = SourceLocation::current());

}  // namespace tools

#endif
