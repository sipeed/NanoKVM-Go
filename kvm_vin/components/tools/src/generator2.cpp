#include <random>

#include <fmt/base.h>
#include <fmt/core.h>
#include <tools/generator.hpp>

namespace tools {

Generator<char> random_character(const std::string& str)
{
    if (str.empty()) {
        throw std::invalid_argument("String cannot be empty");
    }

    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<size_t> dist(0, str.size() - 1);

    while (true) {
        co_yield str[dist(gen)];
    }
}

}  // namespace tools

namespace tools::generator_test {

static tools::Generator<int> fibo()
{
    int a = 0;
    int b = 1;

    co_yield a;
    co_yield b;

    for (;;) {
        int next = a + b;
        co_yield next;
        a = b;
        b = next;
    }
}

int test()
{
    fmt::println("fibo:");
    Generator<int> gen1 = fibo();
    for (int i = 0; i < 10; ++i) {
        fmt::println("{}", gen1.value());
    }
    fmt::println("");

    fmt::println("range:");
    Generator<int> gen2 = range(10, 15);
    for (int i = 0; i < 10; ++i) {
        if (gen2.has_value()) {
            fmt::println("{}", gen2.value());
        } else {
            break;
        }
    }
    fmt::println("");

    fmt::println("range (-):");
    Generator<int> gen2_2 = range(10, -10, -2);
    for (auto i : gen2_2) {
        fmt::println("{}", gen2_2.value());
    }
    fmt::println("");

    fmt::println("range:");
    Generator<int> gen2_3 = range(10);
    for (auto i : gen2_3) {
        fmt::println("{}", gen2_3.value());
    }
    fmt::println("");

    fmt::println("range for");
    auto gen3 = Generator<int>::from(1, 2, 3, 4);
    for (auto i : gen3) {
        fmt::println("{}", i);
    }
    fmt::println("");

    fmt::println("tools test");
    Generator<int>::from(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
        .filter([](auto i) {
            fmt::println("filter: {}", i);
            return i % 2 == 0;
        })
        .map([](auto i) {
            fmt::println("map: {}", i);
            return i * 3;
        })
        .flat_map([](auto i) -> Generator<int> {
            fmt::println("flat_map: {}", i);
            for (int j = 0; j < i; ++j) {
                co_yield j;
            }
        })
        .take(3)
        .for_each([](auto i) { fmt::println("for_each: {}", i); });
    fmt::println("");

    fmt::println("random_range: ");
    random_range(10, 100).take(10).for_each(
        [](auto i) { fmt::print("{} ", i); });
    fmt::println("");

    fmt::println("random_character: ");
    auto gen6 = random_character("01234abcABC").take(10).collect();
    for (auto i : gen6) {
        fmt::print("{} ", i);
    }
    fmt::println("");

    fmt::println("enumerate: ");
    auto gen7 = Generator<int>::from(10, 20, 30);
    for (auto [index, value] : gen7.enumerate()) {
        fmt::println("{} - {}", index, value);
    }
    fmt::println("");

    return 0;
}

}  // namespace tools::generator_test
