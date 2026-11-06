#ifndef __KVM_UI_TOOLS_GENERATORS_HPP__
#define __KVM_UI_TOOLS_GENERATORS_HPP__

#include <coroutine>
#include <exception>
#include <expected>
#include <memory>
#include <optional>
#include <random>
#include <stdexcept>
#include <utility>
#include <vector>

namespace tools {

template<typename T>
struct GeneratorPromise;

template<typename T>
struct Generator {
    using promise_type = GeneratorPromise<T>;

    explicit Generator(std::coroutine_handle<promise_type> handle)
        : _handle(std::move(handle))
    {}

    ~Generator()
    {
        if (this->_handle && !this->_handle.done()) {
            this->_handle.destroy();
        }
    }

    Generator(const Generator&) = delete;
    Generator& operator=(const Generator&) = delete;

    Generator(Generator&& other) noexcept
        : _handle(std::exchange(other._handle, nullptr))
    {}

    Generator& operator=(Generator&& other) noexcept
    {
        if (this != &other) {
            if (_handle)
                _handle.destroy();
            _handle = std::exchange(other._handle, nullptr);
        }
        return *this;
    }

    [[nodiscard]]
    bool has_value()
    {
        if (!this->_handle) {
            return false;
        }
        if (!this->_handle.done() && this->_handle.promise().empty()) {
            this->_handle.resume();
        }
        auto& result = this->_handle.promise();
        if (result.has_value()) {
            return true;
        }
        return false;
    }

    [[nodiscard]]
    T value()
    {
        auto& result = this->_handle.promise();
        if (!this->has_value()) {
            if (this->_handle.promise().has_error()) {
                std::rethrow_exception(result.error());
            } else {
                throw std::runtime_error("Out of range");
            }
        }
        T value = result.value();
        result.clear();
        return value;
    }

    /* iterator */
    struct iterator {
        using iterator_category = std::input_iterator_tag;
        using value_type = T;
        using difference_type = std::ptrdiff_t;
        using pointer = T*;
        using reference = T;

        iterator()
            : gen(nullptr)
            , is_valid(false)
        {}

        explicit iterator(Generator* g)
            : gen(g)
            , is_valid(false)
        {
            if (gen && gen->has_value()) {
                current_value = gen->value();
                is_valid = true;
            }
        }

        reference operator*() const { return current_value; }

        iterator& operator++()
        {
            if (gen && gen->has_value()) {
                current_value = gen->value();
                is_valid = true;
            } else {
                is_valid = false;
            }
            return *this;
        }

        bool operator!=(const iterator& other) const
        {
            return is_valid != other.is_valid;
        }

      private:
        Generator* gen;
        T current_value {};
        bool is_valid;
    };

    iterator begin() { return iterator(this); }
    iterator end() { return iterator(nullptr); }

    /* tools */
    template<typename... Args>
    [[nodiscard]]
    static Generator from(Args... args)
    {
        ((co_yield args), ...);
    }

    template<typename F>
    [[nodiscard]]
    Generator<std::invoke_result_t<F, T>> map(F f)
    {
        auto gen = std::move(*this);
        while (gen.has_value()) {
            co_yield f(gen.value());
        }
    }

    template<typename F>
    [[nodiscard]]
    std::invoke_result_t<F, T> flat_map(F f)
    {
        auto gen = std::move(*this);
        while (gen.has_value()) {
            auto&& generator = f(gen.value());
            while (generator.has_value()) {
                co_yield generator.value();
            }
            if (!generator._handle.done()) {
                generator._handle.destroy();
            }
        }
    }

    template<typename F>
    void for_each(F f)
    {
        while (this->has_value()) {
            f(this->value());
        }
    }

    template<typename R, typename F>
    [[nodiscard]]
    R fold(R initial, F f)
    {
        R acc = initial;
        while (this->has_value()) {
            acc = f(acc, this->value());
        }
        return acc;
    }

    [[nodiscard]]
    T sum()
    {
        T total {};
        while (this->has_value()) {
            total += this->value();
        }
        return total;
    }

    template<typename F>
    [[nodiscard]]
    Generator filter(F f)
    {
        auto gen = std::move(*this);
        while (gen.has_value()) {
            T value = gen.value();
            if (f(value)) {
                co_yield value;
            }
        }
    }

    [[nodiscard]]
    Generator take(size_t n)
    {
        auto gen = std::move(*this);
        for (auto i = decltype(n) {}; i < n; ++i) {
            if (gen.has_value()) {
                co_yield gen.value();
            } else {
                break;
            }
        }
    }

    template<typename F>
    [[nodiscard]]
    Generator take_while(F f)
    {
        auto gen = std::move(*this);
        while (gen.has_value()) {
            T value = gen.value();
            if (f(value)) {
                co_yield value;
            } else {
                break;
            }
        }
    }

    [[nodiscard]]
    std::vector<T> collect()
    {
        std::vector<T> result;
        auto obj = std::move(*this);
        while (obj.has_value()) {
            result.emplace_back(obj.value());
        }
        return result;
    }

    [[nodiscard]]
    Generator<std::pair<size_t, T>> enumerate()
    {
        size_t index = 0;
        auto gen = std::move(*this);
        while (gen.has_value()) {
            T value = gen.value();
            co_yield {index++, value};
        }
    }

  private:
    std::coroutine_handle<promise_type> _handle;
};

template<typename T>
struct GeneratorPromise {
    friend class Generator<T>;
    using Result = std::optional<std::expected<T, std::exception_ptr>>;

    GeneratorPromise() { this->_result = std::nullopt; }

    Generator<T> get_return_object()
    {
        return Generator<T> {
            std::coroutine_handle<GeneratorPromise<T>>::from_promise(*this)};
    }

    std::suspend_always initial_suspend() { return {}; }
    std::suspend_always final_suspend() noexcept { return {}; }
    void unhandled_exception()
    {
        this->_result = std::unexpected(std::current_exception());
    }

    void return_void() { this->_result = std::nullopt; }

    std::suspend_always yield_value(T value)
    {
        this->_result = std::move(value);
        return {};
    }

  private:
    [[nodiscard]]
    bool empty()
    {
        return !this->_result.has_value();
    }

    [[nodiscard]]
    bool has_value()
    {
        return !this->empty() && this->_result.value();  // operator bool()
    }

    [[nodiscard]]
    bool has_error()
    {
        return !this->empty() && !this->_result.value();
    }

    [[nodiscard]]
    T value()
    {
        return this->_result.value().value();
    }

    [[nodiscard]]
    std::exception_ptr error()
    {
        return this->_result.value().error();
    }

    void clear() { this->_result = std::nullopt; }

    Result _result;
};

template<typename T>
[[nodiscard]]
Generator<T> range(T start, T end, T step = 1)
{
    static_assert(std::is_integral_v<T>, "Requires integral type");

    if (step == 0) {
        co_return;
    }

    const bool valid = (step > 0 && end > start) || (step < 0 && end < start);
    if (!valid) {
        co_return;
    }

    T current = start;
    while ((step > 0 && current < end) || (step < 0 && current > end)) {
        co_yield current;
        current += step;
    }
}

template<typename T>
[[nodiscard]]
Generator<T> range(T end)
{
    return range<T>(static_cast<T>(0), end);
}

template<typename T>
[[nodiscard]]
Generator<T> infinite_range(T start = T {0}, T step = T {1})
{
    static_assert(std::is_integral_v<T>, "Requires integral type");

    T current = start;
    while (true) {
        co_yield current;
        current += step;
    }
}

template<typename T>
[[nodiscard]]
Generator<T> random_range(T min, T max)
{
    static_assert(std::is_integral_v<T>, "Must be integral type");
    if (min > max) {
        std::swap(min, max);
    }

    auto state = std::make_unique<
        std::tuple<std::mt19937, std::uniform_int_distribution<T>>>(
        std::random_device {}(), std::uniform_int_distribution<T> {min, max});

    while (true) {
        auto& [engine, dist] = *state;
        co_yield dist(engine);
    }
}

[[nodiscard]]
Generator<char> random_character(const std::string& str);

namespace generator_test {
int test();
}

}  // namespace tools

#endif  // __KVM_UI_TOOLS_GENERATORS_HPP__
