#include <ranges>
#include <vector>

#include <tools/shell.hpp>

namespace tools::shell {

std::string_view shell_trim(std::string_view s)
{
    auto is_space = [](unsigned char c) { return std::isspace(c); };
    auto trimmed = s | std::views::drop_while(is_space) | std::views::reverse
        | std::views::drop_while(is_space) | std::views::reverse;

    if (trimmed.begin() == trimmed.end()) {
        return {};
    }
    return std::string_view(&*trimmed.begin(), std::ranges::distance(trimmed));
}

}  // namespace tools::shell
