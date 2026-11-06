#pragma once

#include <cstdint>
#include <optional>
#include <utility>

#include "lvgl_main/runtime_service.hpp"

namespace lvgl_main
{

// The UI owns synchronization around this value. This policy only determines
// which committed publication may become the next LVGL-thread projection.
class RuntimeStatusMailbox final {
    public:
	bool offer(const RuntimeStatusPublication &publication)
	{
		if (_closed || publication.revision <= _last_applied_revision) {
			return false;
		}
		if (_pending.has_value() &&
		    publication.revision <= _pending->revision) {
			return false;
		}
		_pending = publication;
		return true;
	}

	std::optional<RuntimeStatusPublication> take_next()
	{
		if (_closed || !_pending.has_value()) {
			return std::nullopt;
		}

		RuntimeStatusPublication publication = std::move(*_pending);
		_pending.reset();
		if (publication.revision <= _last_applied_revision) {
			return std::nullopt;
		}
		_last_applied_revision = publication.revision;
		return publication;
	}

	void close()
	{
		_closed = true;
		_pending.reset();
	}

    private:
	std::optional<RuntimeStatusPublication> _pending;
	uint64_t _last_applied_revision { 0 };
	bool _closed { false };
};

} // namespace lvgl_main
