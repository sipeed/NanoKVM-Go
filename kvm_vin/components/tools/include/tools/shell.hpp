#ifndef __KVM_UI_TOOLS_SHELL_HPP__
#define __KVM_UI_TOOLS_SHELL_HPP__

#include <cstdio>
#include <sstream>
#include <string>
#include <string_view>

namespace tools::shell {

std::string_view shell_trim(std::string_view s);

template<typename... Args>
static std::string exec_shell(Args... args)
{
    std::ostringstream cmd_stream;
    ((cmd_stream << args << " "), ...);
    std::string command = cmd_stream.str();
    auto command_view = shell_trim(command);

    FILE* pipe = popen(command_view.data(), "r");
    if (pipe == nullptr) {
        return {};
    }

    char buffer[2048];
    std::string result;
    while (::fgets(buffer, std::size(buffer), pipe) != nullptr) {
        result += buffer;
    }

    int status = pclose(pipe);
    if (status < 0) {
        return {};
    }
    return result;
}

}  // namespace tools::shell

#endif  // __KVM_UI_TOOLS_SHELL_HPP__
