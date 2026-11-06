#include <filesystem>
#include <fstream>
#include <iterator>
#include <string>

#include <catch2/catch_all.hpp>
#include <catch2/catch_test_macros.hpp>
#include <tools/toml_file_parser.hpp>

using namespace tools;

static constexpr std::string_view TOML_PATH = "test_kvm_ui_tomlparser.toml";

static std::string read_toml_file()
{
    std::ifstream ifs(std::string(TOML_PATH), std::ios::binary);
    return std::string(std::istreambuf_iterator<char>(ifs),
                       std::istreambuf_iterator<char>());
}

struct TomlPathFixture {
    TomlPathFixture()
    {
        if (std::filesystem::exists(TOML_PATH)) {
            std::filesystem::remove(TOML_PATH);
        }
        [[maybe_unused]] auto _ = std::ofstream(std::string(TOML_PATH));
        TomlFileParser::set_path(TOML_PATH);
    }
    ~TomlPathFixture()
    {
        if (std::filesystem::exists(TOML_PATH)) {
            std::filesystem::remove(TOML_PATH);
        }
    }
};

TEST_CASE_METHOD(TomlPathFixture,
                 "Test missing int value returns nullopt",
                 "[toml][get]")
{
    auto opt = TomlFileParser::get<int>("test01.intvalue");
    REQUIRE_FALSE(opt.has_value());
}

TEST_CASE_METHOD(TomlPathFixture,
                 "Test set and get int value",
                 "[toml][set][get]")
{
    int value1 = 100;
    SECTION("first write&read")
    {
        bool ok = TomlFileParser::set<int>("test02.intvalue", value1);
        REQUIRE(ok);

        auto opt = TomlFileParser::get<int>("test02.intvalue");
        REQUIRE(opt.has_value());
        REQUIRE(opt.value() == value1);
    }

    SECTION("override&read")
    {
        REQUIRE(TomlFileParser::set<int>("test02.intvalue", value1));

        int value2 = 500;
        bool ok2 = TomlFileParser::set<int>("test02.intvalue", value2);
        REQUIRE(ok2);

        auto opt2 = TomlFileParser::get<int>("test02.intvalue");
        REQUIRE(opt2.has_value());
        REQUIRE(opt2.value() == value2);
    }
}

TEST_CASE_METHOD(TomlPathFixture,
                 "Test set and get nested string value",
                 "[toml][set][get]")
{
    const std::string str = "sipeed";
    bool ok =
        TomlFileParser::set<std::string>("test03.long.long.stringvalue", str);
    REQUIRE(ok);

    auto opt_str =
        TomlFileParser::get<std::string>("test03.long.long.stringvalue");
    REQUIRE(opt_str.has_value());
    REQUIRE(opt_str.value() == str);
}

TEST_CASE_METHOD(TomlPathFixture,
                 "Test set writes trailing newline",
                 "[toml][set]")
{
    REQUIRE(TomlFileParser::set<bool>("snapshot.enable", false));

    const std::string data = read_toml_file();
    REQUIRE_FALSE(data.empty());
    REQUIRE(data.back() == '\n');
}
