#pragma once

#include <algorithm>
#include <array>
#include <cstddef>
#include <cstdint>

namespace lvgl_main::wifi_latency_probe
{

inline constexpr std::size_t kSampleCapacity = 64U;

/** A fixed-capacity reservoir used by the target-only latency probe. */
class SampleSet final {
    public:
	void add(uint64_t value) noexcept;
	[[nodiscard]] uint64_t count() const noexcept;
	[[nodiscard]] uint64_t max() const noexcept;
	[[nodiscard]] uint64_t percentile(uint8_t percent) const noexcept;

    private:
	std::array<uint64_t, kSampleCapacity> _values {};
	std::size_t _size {0U};
	std::size_t _next {0U};
	uint64_t _count {0U};
	uint64_t _max {0U};
};

inline void SampleSet::add(uint64_t value) noexcept
{
	_values[_next] = value;
	_next = (_next + 1U) % _values.size();
	_size = std::min(_size + 1U, _values.size());
	++_count;
	_max = std::max(_max, value);
}

inline uint64_t SampleSet::count() const noexcept
{
	return _count;
}

inline uint64_t SampleSet::max() const noexcept
{
	return _max;
}

inline uint64_t SampleSet::percentile(uint8_t percent) const noexcept
{
	if (_size == 0U) {
		return 0U;
	}
	std::array<uint64_t, kSampleCapacity> sorted {};
	std::copy_n(_values.begin(), _size, sorted.begin());
	std::sort(sorted.begin(), sorted.begin() + static_cast<std::ptrdiff_t>(_size));
	const std::size_t rank =
	    std::max<std::size_t>(1U, (_size * static_cast<std::size_t>(percent) + 99U) / 100U);
	const std::size_t index = std::min(rank - 1U, _size - 1U);
	return sorted[index];
}

} // namespace lvgl_main::wifi_latency_probe
