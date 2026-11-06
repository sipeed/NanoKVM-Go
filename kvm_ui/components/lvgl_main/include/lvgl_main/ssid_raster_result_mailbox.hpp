#pragma once

#include "ui/ssid_prepared_raster.hpp"

#include <mutex>
#include <optional>
#include <utility>

namespace lvgl_main
{

/**
 * One-slot, latest-only bridge from the raster worker to the LVGL thread.
 * It intentionally holds only value types and never requires an LVGL lock.
 */
class SsidRasterResultMailbox {
    public:
	[[nodiscard]] bool offer(ui::SsidPreparedRasterResult result)
	{
		const std::lock_guard lock(_mutex);
		if (_closed) {
			return false;
		}
		_pending = std::move(result);
		return true;
	}

	[[nodiscard]] std::optional<ui::SsidPreparedRasterResult> take()
	{
		const std::lock_guard lock(_mutex);
		auto result = std::move(_pending);
		_pending.reset();
		return result;
	}

	void close()
	{
		const std::lock_guard lock(_mutex);
		_closed = true;
		_pending.reset();
	}

    private:
	std::mutex _mutex;
	std::optional<ui::SsidPreparedRasterResult> _pending;
	bool _closed {false};
};

} // namespace lvgl_main
