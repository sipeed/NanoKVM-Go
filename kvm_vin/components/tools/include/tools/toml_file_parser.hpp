#ifndef __KVM_UI_TOOOLS_TOML_FILE_PARSER_HPP__
#define __KVM_UI_TOOOLS_TOML_FILE_PARSER_HPP__

#include <any>
#include <optional>
#include <string>
#include <string_view>
#include <vector>

#include <tools/tools.hpp>

namespace tools {

class TomlFileParser final {
  public:
    TomlFileParser() = delete;
    ~TomlFileParser() = delete;
    TomlFileParser(const TomlFileParser&) = delete;
    TomlFileParser& operator=(const TomlFileParser&) = delete;
    TomlFileParser(TomlFileParser&&) = delete;
    TomlFileParser& operator=(TomlFileParser&&) = delete;

    static void set_path(std::string_view path);

    template<typename T>
    static std::optional<T> get(const std::string& p);

    template<typename T>
    static bool set(const std::string& p, const T& value);

  private:
    static std::vector<std::string> split_path(const std::string& p);

  private:
    struct Cache {
        std::string path;
        std::any cache;
    };
    inline static tools::Guarded<Cache> _path;
};

extern template std::optional<int> TomlFileParser::get<int>(const std::string&);
extern template std::optional<bool> TomlFileParser::get<bool>(
    const std::string&);
extern template std::optional<std::string> TomlFileParser::get<std::string>(
    const std::string&);

extern template bool TomlFileParser::set<int>(const std::string&, const int&);
extern template bool TomlFileParser::set<bool>(const std::string&, const bool&);
extern template bool TomlFileParser::set<std::string>(const std::string&,
                                                      const std::string&);
namespace toml_file_parser {
int test();
}

}  // namespace tools

#endif  // __KVM_UI_TOOOLS_TOML_FILE_PARSER_HPP__
