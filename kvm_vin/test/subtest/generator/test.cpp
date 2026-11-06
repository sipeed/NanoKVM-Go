#include <catch2/catch_all.hpp>
#include <catch2/catch_test_macros.hpp>
#include <tools/generator.hpp>

using namespace tools;

static cppcoro::generator<int> example(int max)
{
    for (int i = 0; i < max; ++i) {
        co_yield i;
    }
}

TEST_CASE("Generator: from + sum", "[generator][sum]")
{
    using namespace std;
    using namespace Catch;
    auto gen = Generator<int>::from(1, 2, 3, 4);
    auto result = std::move(gen).sum();
    REQUIRE((result == 10));
}

TEST_CASE("Generator: mixed types sum", "[generator][sum][mixed]")
{
    using namespace std;
    using namespace Catch;
    auto gen = Generator<double>::from(1, 2.5, 3);
    auto result = std::move(gen).sum();
    REQUIRE((result == Approx(6.5)));
}

TEST_CASE("Generator: from container and concatenate strings",
          "[generator][from][container]")
{
    using namespace std;
    using namespace Catch;
    vector<string> v {"a", "b", "c"};
    auto gen = Generator<string>::from(v);
    string s;
    for (auto& x : std::move(gen))
        s += x;
    REQUIRE((s == "abc"));
}

TEST_CASE("Generator: from another generator", "[generator][from][generator]")
{
    using namespace std;
    using namespace Catch;
    auto gen = Generator<int>::from(example(5));
    const vector<int> exp = {0, 1, 2, 3, 4};
    vector<int> result;
    for (auto x : std::move(gen)) {
        result.push_back(x);
    }
    REQUIRE((exp == result));
}

TEST_CASE("Generator: take first N elements", "[generator][take]")
{
    using namespace std;
    using namespace Catch;
    auto taken = Generator<int>::from(10, 20, 30, 40).take(3);
    const vector<int> exp = {10, 20, 30};
    vector<int> result;
    for (auto x : std::move(taken)) {
        result.push_back(x);
    }
    REQUIRE((exp == result));
}

TEST_CASE("Generator: take_while predicate holds", "[generator][take_while]")
{
    using namespace std;
    using namespace Catch;
    auto tw = Generator<int>::from(1, 2, 3, 4, 5).take_while([](int x) {
        return x < 4;
    });
    const vector<int> exp = {1, 2, 3};
    vector<int> result;
    for (auto x : std::move(tw)) {
        result.push_back(x);
    }
    REQUIRE((exp == result));
}

TEST_CASE("Generator: map squares", "[generator][map]")
{
    using namespace std;
    using namespace Catch;
    auto m = Generator<int>::from(1, 2, 3).map([](int x) { return x * x; });
    const vector<int> exp = {1, 4, 9};
    vector<int> result;
    for (auto x : std::move(m)) {
        result.push_back(x);
    }
    REQUIRE(exp == result);
}

TEST_CASE("Generator: filter even numbers", "[generator][filter]")
{
    using namespace std;
    using namespace Catch;
    auto f = Generator<int>::from(1, 2, 3, 4, 5).filter([](int x) {
        return x % 2 == 0;
    });
    const vector<int> exp = {2, 4};
    vector<int> result;
    for (auto x : std::move(f)) {
        result.push_back(x);
    }
    REQUIRE(exp == result);
}

TEST_CASE("Generator: flat_map pairs", "[generator][flat_map]")
{
    using namespace std;
    using namespace Catch;
    auto fm = Generator<int>::from(1, 2, 3).flat_map(
        [](int x) { return Generator<int>::from(x, x * 10); });
    const vector<int> exp = {1, 10, 2, 20, 3, 30};
    vector<int> result;
    for (auto x : std::move(fm)) {
        result.push_back(x);
    }
    REQUIRE(exp == result);
}

TEST_CASE("Generator: for_each collects into vector", "[generator][for_each]")
{
    using namespace std;
    using namespace Catch;
    vector<int> out;
    Generator<int>::from(1, 2, 3).for_each([&](int x) { out.push_back(x); });
    REQUIRE(out == vector<int> {1, 2, 3});
}

TEST_CASE("Generator: fold accumulation", "[generator][fold]")
{
    using namespace std;
    using namespace Catch;
    int sum = Generator<int>::from(1, 2, 3, 4).fold(0, [](int a, int b) {
        return a + b;
    });
    REQUIRE(sum == 10);
}

TEST_CASE("Generator: sum convenience method", "[generator][sum]")
{
    using namespace std;
    using namespace Catch;
    REQUIRE(Generator<int>::from(5, 5, 5).sum() == 15);
}

TEST_CASE("Generator: enumerate indices and values", "[generator][enumerate]")
{
    using namespace std;
    using namespace Catch;
    auto gen = Generator<int>::from(10, 20, 30).enumerate();
    auto result = move(gen).to_vector();
    REQUIRE((result.size() == 3));
    REQUIRE((result[0] == make_pair<size_t, int>(0, 10)));
    REQUIRE((result[1] == make_pair<size_t, int>(1, 20)));
    REQUIRE((result[2] == make_pair<size_t, int>(2, 30)));
}

TEST_CASE("Generator: zip two generators of equal length", "[generator][zip]")
{
    using namespace std;
    using namespace Catch;
    auto g1 = Generator<int>::from(1, 2, 3);
    auto g2 = Generator<string>::from("a", "b", "c");
    auto zipped = move(g1).zip(move(g2));
    auto out = move(zipped).to_vector();
    vector<std::pair<int, string>> exp = {
        {1, "a"},
        {2, "b"},
        {3, "c"}
    };
    REQUIRE((exp == out));
}

TEST_CASE("Generator: zip_longest with default", "[generator][zip_longest]")
{
    using namespace std;
    using namespace Catch;
    auto g1 = Generator<int>::from(1, 2);
    auto g2 = Generator<int>::from(10, 20, 30);
    auto zl = move(g1).zip_longest(move(g2), optional<int> {-1});
    auto out = move(zl).to_vector();
    vector<std::pair<int, optional<int>>> exp = {
        {1, 10},
        {2, 20},
        {0, -1}
    };
    REQUIRE((exp == out));
}

TEST_CASE("Generator: cycle and take N", "[generator][cycle]")
{
    using namespace std;
    using namespace Catch;
    auto base = Generator<char>::from('x', 'y');
    auto cycled = move(base).cycle().take(5);
    string s;
    for (char c : move(cycled))
        s.push_back(c);
    REQUIRE((s == "xyxyx"));
}

TEST_CASE("Generator: repeat value N times", "[generator][repeat]")
{
    using namespace std;
    using namespace Catch;
    auto rep = Generator<string>::repeat("hi", 3);
    auto out = move(rep).to_vector();
    vector<string> exp = {"hi", "hi", "hi"};
    REQUIRE((exp == out));
}

TEST_CASE("Generator: drop_while predicate till false",
          "[generator][drop_while]")
{
    using namespace std;
    using namespace Catch;
    auto seq = Generator<int>::from(1, 2, 3, 4, 1, 2);
    auto dropped = move(seq).drop_while([](int x) { return x < 3; });
    auto out = move(dropped).to_vector();
    vector<int> exp = {3, 4, 1, 2};
    REQUIRE((exp == out));
}

TEST_CASE("Generator: scan accumulate sequence", "[generator][scan]")
{
    using namespace std;
    using namespace Catch;
    auto seq = Generator<int>::from(1, 2, 3, 4);
    auto scanned = move(seq).scan(0, [](int acc, int x) { return acc + x; });
    auto out = move(scanned).to_vector();
    vector<int> exp = {1, 3, 6, 10};
    REQUIRE((exp == out));
}

TEST_CASE(
    "Generator: terminal operations (to_vector, find, any, all, min, max)",
    "[generator][terminal]")
{
    using namespace std;
    using namespace Catch;
    auto seq = Generator<int>::from(5, 2, 8, 3);
    REQUIRE(move(seq).to_vector() == vector<int> {5, 2, 8, 3});

    auto seq2 = Generator<int>::from(5, 2, 8, 3);
    auto found = move(seq2).find([](int x) { return x > 6; });
    REQUIRE(found.has_value());
    REQUIRE(found.value() == 8);

    auto seq3 = Generator<int>::from(5, 2, 8, 3);
    REQUIRE(move(seq3).any([](int x) { return x == 2; }));

    auto seq4 = Generator<int>::from(5, 5);
    REQUIRE(move(seq4).all([](int x) { return x == 5; }));

    auto seq5 = Generator<int>::from(5, 2, 8, 3);
    REQUIRE(move(seq5).min() == 2);

    auto seq6 = Generator<int>::from(5, 2, 8, 3);
    REQUIRE(move(seq6).max() == 8);
}

TEST_CASE("Generator: conversion via iteration", "[generator][conversion]")
{
    using namespace std;
    using namespace Catch;
    auto cv = Generator<int>::from(example(4));
    const vector<int> exp = {0, 1, 2, 3};
    vector<int> result;
    for (auto x : move(cv)) {
        result.push_back(x);
    }
    REQUIRE((exp == result));
}

TEST_CASE("Generator: range [0, end)", "[generator][range]")
{
    using namespace std;
    using namespace Catch;
    auto gen = Generator<int>::range(5);
    const vector<int> exp = {0, 1, 2, 3, 4};
    vector<int> result;
    for (auto x : move(gen)) {
        result.push_back(x);
    }
    REQUIRE((exp == result));
}

TEST_CASE("Generator: range [start, end) with step", "[generator][range]")
{
    using namespace std;
    using namespace Catch;
    auto gen = Generator<int>::range(5, 12, 2);
    const vector<int> exp = {5, 7, 9, 11};
    vector<int> result;
    for (auto x : move(gen)) {
        result.push_back(x);
    }
    REQUIRE((exp == result));
}

TEST_CASE("Generator: random integral values in [min, max)",
          "[generator][random]")
{
    using namespace std;
    using namespace Catch;
    auto gen = Generator<int>::randoms(10, 20);
    auto index = Generator<int>::range(100);
    for (auto& [_, value] : move(index).zip(move(gen))) {
        REQUIRE(value >= 10);
        REQUIRE(value < 20);
    }
}

TEST_CASE("Generator: random char from string pool", "[generator][random]")
{
    using namespace std;
    using namespace Catch;
    const string pool = "abcde";
    auto gen = Generator<char>::randoms(pool);
    int count = 0;
    for (auto c : move(gen)) {
        if (count++ > 100)
            break;
        REQUIRE(pool.find(c) != string::npos);
    }
}

TEST_CASE("Generator: char/string to string", "[generator][to_string]")
{
    auto gen = Generator<char>::from('a', 'b', 'c');
    auto result = std::move(gen).to_string();
    REQUIRE(result == "abc");
}
