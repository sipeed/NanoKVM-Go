#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

#include "ui/scona.hpp"
#include "ui/ssid_prepared_raster.hpp"

namespace ui
{

class SsidRasterMarquee;

// Fixed SSID key/value row. The value is a copied cache image, an image-backed
// runtime-font marquee, or a compiled-font fallback label. It never exposes a
// runtime font through a visible label.
class SsidInfoRow final : public Component {
    public:
	explicit SsidInfoRow(lv_obj_t *parent);
	~SsidInfoRow() override;

	void create() override;
	void destroy() override;

	// Copies a compatible ARGB8888 cached source into the visible value canvas.
	// The caller retains ownership of source and may release it after this call.
	bool set_image_source(const lv_draw_buf_t *source);
	// Converts worker-owned logical pixels to an LVGL image only on the UI
	// thread. The worker never sees this API or an lv_draw_buf_t.
	bool set_prepared_raster(const SsidPreparedRaster &raster,
				 bool start_scroll);

	// Selects the standard compiled-font fallback presentation.
	bool set_fallback_text(const char *text);
	void set_fallback_scroll_enabled(bool enabled);
	void clear_raster_marquee() noexcept;
	void pause_ssid_scroll() noexcept;
	void resume_ssid_scroll() noexcept;

	bool has_image_source() const noexcept
	{
		return _image_rendered;
	}

	static constexpr uint16_t WIDTH = SConA::WIDTH;
	static constexpr uint16_t HEIGHT = SConA::HEIGHT;
	static constexpr uint16_t KEY_HORIZONTAL_INSET = 12;
	static constexpr uint16_t VALUE_X = 68;
	static constexpr uint16_t VALUE_RIGHT_INSET = 12;
	static constexpr uint16_t VALUE_WIDTH =
		WIDTH - VALUE_X - VALUE_RIGHT_INSET;

    private:
	lv_obj_t *_key_label = nullptr;
	lv_obj_t *_value_canvas = nullptr;
	lv_obj_t *_fallback_label = nullptr;
	std::vector<uint8_t> _canvas_storage;
	std::string _fallback_text { "--" };
	std::unique_ptr<SsidRasterMarquee> _raster_marquee;
	bool _image_rendered = false;
	bool _fallback_scroll_enabled = false;
	bool _fallback_scroll_paused = false;

	bool canvas_available() const noexcept;
	bool source_is_compatible(const lv_draw_buf_t *source) const noexcept;
	void clear_canvas();
	void apply_fallback_scroll_mode() noexcept;
	void stop_fallback_scroll() noexcept;
	bool install_prepared_raster_now(const SsidPreparedRaster &raster,
					 bool start_scroll);

	friend class SsidInfoRowTestAccess;
	friend class WifiPageTestAccess;
};

} // namespace ui
