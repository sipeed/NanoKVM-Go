#ifndef __LIBKVM_GLOBAL_PARAMETERS_TOML_DEFAULT_HPP__
#define __LIBKVM_GLOBAL_PARAMETERS_TOML_DEFAULT_HPP__

#include <string_view>

namespace global_para::toml_default {

constexpr std::string_view CONFIG_FILE_PATH = "/etc/kvm/libkvm.toml";

constexpr std::string_view GLOBAL_LOG_LEVEL_PATH = "log.level";
constexpr std::string_view GLOBAL_LOG_LEVEL_DEF = "error";

}  // namespace global_para::toml_default

#endif  // __LIBKVM_GLOBAL_PARAMETERS_TOML_DEFAULT_HPP__
