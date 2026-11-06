#include <cassert>
#include <exception>
#include <mutex>
#include <sstream>
#include <string_view>
#include <type_traits>

#include <spdlog/common.h>
#include <spdlog/spdlog.h>  // only for test()
#include <toml++/impl/node_view.hpp>
#include <toml++/impl/parser.hpp>
#include <toml++/toml.hpp>
#include <tools/toml_file_parser.hpp>

#include "tools/tools.hpp"

namespace tools {

constexpr bool DEBUG_MODE = false;

template<typename Callable,
         typename... Args,
         typename RT = std::invoke_result_t<Callable, Args...>>
static RT on_debug(Callable&& func, Args&&... args) noexcept
{
    if constexpr (DEBUG_MODE) {
        return std::forward<Callable>(func)(std::forward<Args>(args)...);
    } else {
        if constexpr (std::is_void_v<RT>) {
            return;
        } else {
            return RT {};
        }
    }
}

void TomlFileParser::set_path(std::string_view path)
{
    if constexpr (DEBUG_MODE) {
        spdlog::set_level(spdlog::level::debug);
    }

    std::unique_lock lock(_path.mutex);
    _path.data.path = path;

    try {
        auto config = toml::parse_file(_path.data.path);
        _path.data.cache = std::move(config);
        on_debug(
            [](std::string_view p) {
                spdlog::debug("{} load cache from {} finish", tools::tag(), p);
            },
            path);
    } catch (...) {
        _path.data.cache = {};
        on_debug(
            [](std::string_view p) {
                spdlog::error("{} load cache from {} failed", tools::tag(), p);
            },
            path);
    }
}

template<typename T>
std::optional<T> TomlFileParser::get(const std::string& p)
{
    std::shared_lock lock(_path.mutex);

    if (_path.data.cache.type() != typeid(toml::table)) {
        on_debug([]() { spdlog::error("{} cache expiration", tools::tag()); });
        return std::nullopt;
    }

    try {
        auto& config = std::any_cast<toml::table&>(_path.data.cache);
        toml::node_view<toml::node> node {config};
        const auto parts = split_path(p);
        for (const auto& part : parts) {
            node = node[part];
            if (!node) {
                on_debug(
                    [](std::string_view path) {
                        spdlog::debug(
                            "{} invalid path: {}", tools::tag(), path);
                    },
                    p);
                return std::nullopt;
            }
        }

        if constexpr (std::is_same_v<T, toml::table>) {
            return node.as_table() ? std::optional {*node.as_table()}
                                   : std::nullopt;
        } else if constexpr (std::is_same_v<T, toml::array>) {
            return node.as_array() ? std::optional {*node.as_array()}
                                   : std::nullopt;
        } else {
            auto val = node.value<T>();
            return val ? std::optional {*val} : std::nullopt;
        }
    } catch (const std::exception& e) {
        on_debug(
            [](const std::exception& ex) {
                spdlog::error("{} exception: {}", tools::tag(), ex.what());
            },
            e);
        return std::nullopt;
    }
}

template<typename T>
bool TomlFileParser::set(const std::string& p, const T& value)
{
    std::unique_lock lock(_path.mutex);

    if (_path.data.cache.type() != typeid(toml::table)) {
        on_debug([]() { spdlog::error("{} cache expiration", tools::tag()); });
        return false;
    }

    try {
        toml::table config = std::any_cast<toml::table>(_path.data.cache);
        auto parts = split_path(p);

        toml::table* current = &config;
        for (size_t i = 0; i < parts.size() - 1; ++i) {
            if (!current->contains(parts[i])) {
                current->insert(parts[i], toml::table {});
            }
            current = current->at(parts[i]).as_table();
            if (!current)
                return false;
        }

        current->insert_or_assign(parts.back(), value);

        std::ostringstream oss;
        oss << config;
        std::string content = oss.str();
        if (content.empty() || content.back() != '\n') {
            content.push_back('\n');
        }

        std::ofstream ofs(_path.data.path);
        ofs << content;

        _path.data.cache = config;
        return true;
    } catch (const std::exception& e) {
        on_debug(
            [](const std::exception& _e) {
                spdlog::error("{} exception: {}", tools::tag(), _e.what());
            },
            e);
        return false;
    }

    return false;
}

std::vector<std::string> TomlFileParser::split_path(const std::string& p)
{
    std::vector<std::string> parts;
    std::stringstream ss(p);
    std::string part;
    while (std::getline(ss, part, '.')) {
        parts.emplace_back(std::move(part));
    }
    return parts;
}

template std::optional<int> TomlFileParser::get<int>(const std::string& path);
template std::optional<bool> TomlFileParser::get<bool>(const std::string& path);
template std::optional<std::string> TomlFileParser::get<std::string>(
    const std::string& path);

template bool TomlFileParser::set<int>(const std::string& path, const int& val);
template bool TomlFileParser::set<bool>(const std::string& path,
                                        const bool& val);
template bool TomlFileParser::set<std::string>(const std::string& path,
                                               const std::string& val);

int toml_file_parser::test()
{
    constexpr std::string_view TOML_PATH = "/root/test.toml";

    tools::TomlFileParser::set_path(TOML_PATH);

    /* test01_intvalue NOT exists, will return std::nullopt */
    const auto test01_intvalue =
        tools::TomlFileParser::get<int>("test01.intvalue");
    assert(test01_intvalue == std::nullopt);
    if (test01_intvalue == std::nullopt) {
        spdlog::debug("test01.intvalue NOT FOUND... pass");
    }

    int value = 100;
    const auto test02_intvalue_ret =
        tools::TomlFileParser::set<int>("test02.intvalue", value);
    assert(test02_intvalue_ret == true);
    if (test02_intvalue_ret) {
        spdlog::debug("test02.intvalue write... pass");
    }

    const auto test02_intvalue =
        tools::TomlFileParser::get<int>("test02.intvalue");
    assert(test02_intvalue == value);
    if (test02_intvalue.value() == value) {
        spdlog::debug("test02.intvalue read: {}...pass",
                      test02_intvalue.value());
    }

    value = 500;
    const auto test02_intvalue_ret2 =
        tools::TomlFileParser::set<int>("test02.intvalue", value);
    assert(test02_intvalue_ret == true);
    if (test02_intvalue_ret2) {
        spdlog::debug("test02.intvalue write... pass");
    }

    const auto test02_intvalue2 =
        tools::TomlFileParser::get<int>("test02.intvalue");
    assert(test02_intvalue2.value() == value);
    if (test02_intvalue2.value() == value) {
        spdlog::debug("test02.intvalue read: {}...pass",
                      test02_intvalue2.value());
    }

    const auto test03_long_path_string_ret =
        tools::TomlFileParser::set<std::string>("test03.long.long.stringvalue",
                                                "sipeed");
    assert(test03_long_path_string_ret == true);
    if (test03_long_path_string_ret) {
        spdlog::debug("test03.long.long.stringvalue write...pass");
    }

    const auto test03_long_path_string =
        tools::TomlFileParser::get<std::string>("test03.long.long.stringvalue");
    assert(test03_long_path_string.value() == "sipeed");
    if (test03_long_path_string.value() == "sipeed") {
        spdlog::debug("test03.long.long.stringvalue read: {}...pass",
                      test03_long_path_string.value());
    }

    return 0;
}

}  // namespace tools
