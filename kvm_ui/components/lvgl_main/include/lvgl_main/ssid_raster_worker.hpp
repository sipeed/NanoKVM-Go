#pragma once

#include "ui/ssid_prepared_raster.hpp"

#include <functional>
#include <memory>
#include <string>

namespace lvgl_main
{

/**
 * Configuration contains native filesystem paths only.  The worker deliberately
 * does not receive lv_font_t or any other LVGL object.
 */
struct SsidRasterWorkerConfig {
	std::string base_font_path;
	std::string cjk_font_path;
	uint16_t font_size_px {20};
	uint32_t max_font_bytes {64U * 1024U * 1024U};
	uint32_t max_text_bytes {512};
};

class SsidRasterWorker {
    public:
	using CancelPredicate = std::function<bool()>;
	using Completion = std::function<void(ui::SsidPreparedRasterResult)>;

	/**
	 * An optional test seam.  Production callers leave this empty, causing the
	 * private stb_truetype implementation to rasterize the request.
	 */
	using TestRasterizer = std::function<ui::SsidPreparedRasterResult(
	    const ui::SsidRasterRequest &, const CancelPredicate &)>;

	SsidRasterWorker(SsidRasterWorkerConfig config, Completion completion,
			 TestRasterizer test_rasterizer = {});
	~SsidRasterWorker();

	SsidRasterWorker(const SsidRasterWorker &) = delete;
	SsidRasterWorker &operator=(const SsidRasterWorker &) = delete;

	/**
	 * Replaces the pending request.  A request already being rasterized sees
	 * cancellation at glyph/row boundaries and cannot publish after replacement.
	 */
	[[nodiscard]] bool submit(ui::SsidRasterRequest request);

	/**
	 * Invalidates any queued/in-flight result for a page.  It is safe to call
	 * during UI teardown before closing the worker.
	 */
	void invalidate_page(uint64_t page_token) noexcept;

	/**
	 * Stops the worker and joins its thread.  Completion is never invoked after
	 * this function returns.
	 */
	void close() noexcept;

    private:
	struct Impl;
	std::unique_ptr<Impl> _impl;
};

} // namespace lvgl_main
