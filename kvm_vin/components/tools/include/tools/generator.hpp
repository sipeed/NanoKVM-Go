#ifndef __KVM_UI_TOOLS_GENERATORS_HPP__
#define __KVM_UI_TOOLS_GENERATORS_HPP__

#include <algorithm>
#include <cassert>
#include <concepts>
#include <cstddef>
#include <memory>
#include <optional>
#include <random>
#include <ranges>
#include <type_traits>
#include <utility>

#include <cppcoro/generator.hpp>
#include <fmt/base.h>
#include <fmt/core.h>

namespace tools {

// NOLINTBEGIN
template<typename T>
class Generator {
  public:
    Generator() = delete;
    explicit Generator(cppcoro::generator<T>&& gen) noexcept
        : _gen(std::move(gen))
    {}

    ~Generator() = default;

    Generator(const Generator&) = delete;
    Generator& operator=(const Generator&) = delete;

    Generator(Generator&& other) noexcept = default;
    Generator& operator=(Generator&& other) noexcept = default;

    // Generator(Generator&& other) noexcept
    //     : _gen(std::exchange(other._gen, {}))
    // {}
    //
    // Generator& operator=(Generator&& other) noexcept
    // {
    //     if (this != &other) {
    //         this->_gen = std::exchange(other._gen, {});
    //     }
    //     return *this;
    // }

    // Range [start, end)
    [[nodiscard]]
    static Generator range(T start, T end, T step = T {1})
        requires std::default_initializable<T> && std::copyable<T>
        && requires(T a, T b) {
               { a + b } -> std::convertible_to<T>;
           }
    {
        return Generator {range_subseq(start, end, step)};
    }

    // Range [0, end) step=1
    [[nodiscard]]
    static Generator range(T end)
        requires std::default_initializable<T> && std::copyable<T>
        && requires(T a, T b) {
               { a + b } -> std::convertible_to<T>;
           }
    {
        return Generator {range_subseq(T {0}, end, T {1})};
    }

    // Infinite Range
    [[nodiscard]]
    static Generator infinite_range(T start = T {0}, T step = T {1})
        requires std::is_integral_v<T>
    {
        return Generator {infinite_range_subseq(start, step)};
    }

    // Randoms for integral
    [[nodiscard]]
    static Generator randoms(T min, T max)
        requires std::is_integral_v<T>
    {
        return Generator {random_subseq(min, max)};
    }

    // Ramdons for char
    [[nodiscard]]
    static Generator randoms(const std::string str)
        requires std::same_as<T, char>
    {
        return Generator {random_subseq(str)};
    }

    // From cppcoro::generator
    [[nodiscard]]
    static Generator from(cppcoro::generator<T>&& gen)
    {
        return Generator {std::move(gen)};
    }

    // Construct from values
    template<typename... Args>
        requires(sizeof...(Args) > 0) && (std::convertible_to<Args, T> && ...)
    [[nodiscard]] static Generator from(Args&&... args)
    {
        std::vector<T> values;
        values.reserve(sizeof...(Args));
        (values.push_back(static_cast<T>(std::forward<Args>(args))), ...);
        return Generator {from_subseq(values)};
    }

    // Construct from range
    template<std::ranges::input_range R>
    [[nodiscard]]
    static Generator<std::ranges::range_value_t<R>> from(R&& range)
    {
        using U = std::ranges::range_value_t<R>;
        cppcoro::generator<U> seq = from_range_subseq(std::forward<R>(range));
        return Generator<U> {std::move(seq)};
    }

    // Map
    template<typename F, typename RT = std::invoke_result_t<F, T>>
    [[nodiscard]]
    Generator<RT> map(F f) &&
    {
        return Generator<RT> {map_subseq(std::move(this->_gen), f)};
    }

    // Flat map
    template<typename F, typename SubGen = std::invoke_result_t<F, T>>
        requires std::ranges::input_range<SubGen>
    [[nodiscard]]
    Generator<std::ranges::range_value_t<SubGen>> flat_map(F f) &&
    {
        return Generator {flat_map_subseq(std::move(this->_gen), f)};
    }

    // For each
    template<typename F>
    void for_each(F f) &&
    {
        for_each_subseq(std::move(this->_gen), f);
    }

    // Fold
    template<typename R, typename F>
    [[nodiscard]]
    R fold(R initial, F f) &&
    {
        return fold_subseq(std::move(this->_gen), initial, f);
    }

    // Filter
    template<typename F>
    [[nodiscard]]
    Generator filter(F f) &&
    {
        return Generator {filter_subseq(std::move(this->_gen), f)};
    }

    // Take
    [[nodiscard]]
    Generator take(std::size_t n) &&
    {
        auto g = std::move(this->_gen);
        auto gen = take_subseq(std::move(g), n);
        return Generator {std::move(gen)};
    }

    // Take while
    template<typename F>
    [[nodiscard]]
    Generator take_while(F f) &&
    {
        return Generator {take_while_subseq(std::move(this->_gen), f)};
    }

    // Sum
    [[nodiscard]]
    T sum() &&
        requires std::default_initializable<T> && requires(T a, T b) {
            { a + b } -> std::convertible_to<T>;
        }
    {
        T total {};
        for (auto&& value : this->_gen) {
            total += value;
        }
        return total;
    }

    // Enumerate
    [[nodiscard]]
    Generator<std::pair<std::size_t, T>> enumerate() &&
    {
        return Generator<std::pair<std::size_t, T>> {
            enumerate_subseq(std::move(this->_gen))};
    }

    // Zip
    template<typename U>
    [[nodiscard]]
    Generator<std::pair<T, U>> zip(Generator<U>&& other) &&
    {
        return Generator<std::pair<T, U>> {
            zip_subseq(std::move(this->_gen), std::move(other._gen))};
    }

    // Zip Longest
    template<typename U>
    [[nodiscard]]
    Generator<std::pair<T, std::optional<U>>> zip_longest(
        Generator<U>&& other, std::optional<U> fill = std::nullopt) &&
    {
        return Generator<std::pair<T, std::optional<U>>> {zip_longest_subseq(
            std::move(this->_gen), std::move(other._gen), fill)};
    }

    // Cycle
    [[nodiscard]]
    Generator cycle() &&
    requires(std::copyable<T>) {
        /***/
        return Generator {cycle_subseq(std::move(this->_gen))};
    }

    // Repeat
    [[nodiscard]]
    static Generator repeat(T value, std::size_t count = std::numeric_limits<std::size_t>::max())
    {
        return Generator {repeat_subseq(value, count)};
    }

    // Drop While
    template<typename F>
    [[nodiscard]]
    Generator drop_while(F f) &&
    {
        return Generator {drop_while_subseq(std::move(this->_gen), f)};
    }

    // Scan
    template<typename R2 = T, typename F>
    [[nodiscard]]
    Generator<R2> scan(R2 initial, F f) &&
    {
        return Generator<R2> {
            scan_subseq<R2>(std::move(this->_gen), initial, f)};
    }

    // To Vector
    [[nodiscard]]
    std::vector<T> to_vector() &&
    {
        std::vector<T> out;
        for (auto&& v : this->_gen) {
            out.push_back(v);
        }
        return out;
    }

    // To String
    template<typename U = T>
        requires(std::same_as<T, char> || std::same_as<T, std::string>)
    [[nodiscard]]
    std::string to_string() &&
    {
        std::string result;
        for (auto&& v : this->_gen) {
            result += v;
        }
        return result;
    }

    // Find
    [[nodiscard]]
    std::optional<T> find(std::function<bool(const T&)> pred) &&
    {
        for (auto&& v : this->_gen) {
            if (pred(v)) {
                return v;
            }
        }
        return std::nullopt;
    }

    // Any
    [[nodiscard]]
    bool any(std::function<bool(const T&)> pred) &&
    {
        for (auto&& v : this->_gen) {
            if (pred(v)) {
                return true;
            }
        }
        return false;
    }

    // All
    [[nodiscard]]
    bool all(std::function<bool(const T&)> pred) &&
    {
        for (auto&& v : this->_gen) {
            if (!pred(v)) {
                return false;
            }
        }
        return true;
    }

    // Min
    [[nodiscard]]
    T min() &&
    {
        auto it = this->_gen.begin();
        if (it == this->_gen.end()) {
            throw std::runtime_error("min of empty generator");
        }
        T m = *it;
        ++it;
        for (; it != this->_gen.end(); ++it) {
            if (*it < m) {
                m = *it;
            }
        }
        return m;
    }

    // Max
    [[nodiscard]]
    T max() &&
    {
        auto it = this->_gen.begin();
        if (it == this->_gen.end()) {
            throw std::runtime_error("max of empty generator");
        }
        T m = *it;
        ++it;
        for (; it != this->_gen.end(); ++it) {
            if (m < *it) {
                m = *it;
            }
        }
        return m;
    }

    // Iterators
    [[nodiscard]]
    auto begin() &
    {
        return this->_gen.begin();
    }

    [[nodiscard]]
    auto end() &
    {
        return this->_gen.end();
    }

    [[nodiscard]]
    auto begin() &&
    {
        return this->_gen.begin();
    }

    [[nodiscard]]
    auto end() &&
    {
        return this->_gen.end();
    }

    // Convert to cppcoro::generator
    [[nodiscard]]
    explicit operator cppcoro::generator<T>() &&
    {
        return std::move(this->_gen);
    }

  private:
    // Subsequence implementations

    [[nodiscard]]
    static cppcoro::generator<T> infinite_range_subseq(T start, T step)
    {
        T current = start;
        while (true) {
            co_yield current;
            current += step;
        }
    }

    [[nodiscard]]
    static cppcoro::generator<T> random_subseq(T min, T max)
    {
        T mi = min;
        T ma = max;
        if (mi > ma) {
            std::swap(mi, ma);
        }

        auto state = std::make_unique<
            std::tuple<std::mt19937, std::uniform_int_distribution<T>>>(
            std::random_device {}(),
            std::uniform_int_distribution<int> {mi, ma - 1});
        while (true) {
            auto& [engine, dist] = *state;
            co_yield dist(engine);
        }
    }

    [[nodiscard]]
    static cppcoro::generator<char> random_subseq(const std::string str)
    {
        if (str.empty()) {
            co_return;
        }

        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<std::size_t> dist(0, str.size() - 1);

        while (true) {
            char value = str[dist(gen)];
            co_yield value;
        }
    }

    [[nodiscard]]
    static cppcoro::generator<T> range_subseq(T start, T end, T step)
    {
        if (step == T {}) {
            co_return;
        }

        const bool valid =
            (step > 0 && end > start) || (step < 0 && end < start);
        if (!valid) {
            co_return;
        }

        T current = start;
        while ((step > 0 && current < end) || (step < 0 && current > end)) {
            co_yield current;
            current += step;
        }
    }

    [[nodiscard]]
    static cppcoro::generator<T> take_subseq(cppcoro::generator<T> gen,
                                             std::size_t n)
    {
        std::size_t count = 0;
        for (auto&& value : gen) {
            if (count++ < n) {
                co_yield value;
            } else {
                co_return;
            }
        }
    }

    [[nodiscard]]
    static cppcoro::generator<T> from_subseq(std::vector<T> values)
    {
        for (auto& v : values) {
            co_yield v;
        }
    }

    template<std::ranges::input_range R>
    [[nodiscard]]
    static cppcoro::generator<std::ranges::range_value_t<R>> from_range_subseq(
        R&& range)
    {
        for (auto&& v : range) {
            co_yield v;
        }
    }

    template<typename F, typename RT = std::invoke_result_t<F, T>>
    [[nodiscard]]
    static cppcoro::generator<RT> map_subseq(cppcoro::generator<T> gen, F f)
    {
        for (auto&& i : gen) {
            co_yield f(i);
        }
    }

    template<typename F, typename SubGen = std::invoke_result_t<F, T>>
        requires std::ranges::input_range<SubGen>
    [[nodiscard]]
    static cppcoro::generator<std::ranges::range_value_t<SubGen>>
    flat_map_subseq(cppcoro::generator<T> g, F f)
    {
        for (auto&& i : g) {
            for (auto&& j : f(i))
                co_yield j;
        }
    }

    template<typename F>
    static void for_each_subseq(cppcoro::generator<T> gen, F f)
    {
        for (auto&& i : gen) {
            f(i);
        }
    }

    template<typename R, typename F>
    [[nodiscard]]
    static R fold_subseq(cppcoro::generator<T> gen, R initial, F f)
    {
        R acc = initial;
        for (auto&& i : gen) {
            acc = f(acc, i);
        }
        return acc;
    }

    template<typename F>
    [[nodiscard]]
    static cppcoro::generator<T> filter_subseq(cppcoro::generator<T> gen, F f)
    {
        for (auto&& i : gen) {
            if (f(i)) {
                co_yield i;
            }
        }
    }

    template<typename F>
    [[nodiscard]]
    static cppcoro::generator<T> take_while_subseq(cppcoro::generator<T> gen,
                                                   F f)
    {
        for (auto&& i : gen) {
            if (f(i)) {
                co_yield i;
            } else {
                co_return;
            }
        }
    }

    [[nodiscard]]
    static cppcoro::generator<std::pair<std::size_t, T>> enumerate_subseq(
        cppcoro::generator<T> gen)
    {
        std::size_t index = 0;
        for (auto&& i : gen) {
            co_yield std::make_pair(index++, i);
        }
    }

    template<typename U>
    [[nodiscard]]
    static cppcoro::generator<std::pair<T, U>> zip_subseq(
        cppcoro::generator<T> gen1, cppcoro::generator<U> gen2)
    {
        auto it1 = gen1.begin();
        const auto end1 = gen1.end();
        auto it2 = gen2.begin();
        const auto end2 = gen2.end();

        while (it1 != end1 && it2 != end2) {
            co_yield {*it1, *it2};
            ++it1;
            ++it2;
        }
    }

    template<typename U>
    [[nodiscard]]
    static cppcoro::generator<std::pair<T, std::optional<U>>>
    zip_longest_subseq(cppcoro::generator<T> gen1,
                       cppcoro::generator<U> gen2,
                       std::optional<U> fill)
    {
        auto it1 = gen1.begin();
        const auto end1 = gen1.end();
        auto it2 = gen2.begin();
        const auto end2 = gen2.end();

        for (; it1 != end1 || it2 != end2;) {
            if (it1 != end1 && it2 != end2) {
                co_yield {*it1, std::optional<U> {*it2}};
                ++it1;
                ++it2;
            } else {
                co_yield {typename std::decay<T>::type {}, fill};
                if (it1 != end1) {
                    ++it1;
                }
                if (it2 != end2) {
                    ++it2;
                }
            }
        }
    }

    [[nodiscard]]
    static cppcoro::generator<T> cycle_subseq(cppcoro::generator<T> gen)
    {
        std::vector<T> saved;
        for (auto&& v : gen) {
            co_yield v;
            saved.push_back(v);
        }
        while (!saved.empty()) {
            for (auto&& v : saved) {
                co_yield v;
            }
        }
    }

    [[nodiscard]]
    static cppcoro::generator<T> repeat_subseq(T value, std::size_t count)
    {
        for (size_t i = 0; i < count; ++i) {
            co_yield value;
        }
    }

    template<typename F>
    [[nodiscard]]
    static cppcoro::generator<T> drop_while_subseq(cppcoro::generator<T> gen,
                                                   F f)
    {
        bool dropping = true;
        for (auto&& v : gen) {
            if (dropping && f(v)) {
                continue;
            }
            dropping = false;
            co_yield v;
        }
    }

    template<typename R2, typename F>
    [[nodiscard]]
    static cppcoro::generator<R2> scan_subseq(cppcoro::generator<T> gen,
                                              R2 init,
                                              F f)
    {
        R2 acc = init;
        for (auto&& v : gen) {
            acc = f(acc, v);
            co_yield acc;
        }
    }

  public:
    cppcoro::generator<T> _gen;
};
}  // namespace tools

template<typename T>
using Gen = tools::Generator<T>;

#endif  // __KVM_UI_TOOLS_GENERATORS_HPP__
