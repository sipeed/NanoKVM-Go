#include "ui/ssid_info_row.hpp"

#include <cstddef>
#include <new>

#include <spdlog/spdlog.h>

#include "ui/ssid_raster_marquee.hpp"
#include "ui/wifi_ssid_font.hpp"

namespace ui
{

namespace
{

constexpr lv_area_t VALUE_DRAW_AREA {
	.x1 = 0,
	.y1 = 0,
	.x2 = SsidInfoRow::VALUE_WIDTH - 1,
	.y2 = SsidInfoRow::HEIGHT - 1,
};

void configure_noninteractive(lv_obj_t *object)
{
	if (object == nullptr) {
		return;
	}
	lv_obj_clear_flag(object, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(object, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(object, LV_OBJ_FLAG_IGNORE_LAYOUT);
}

} // namespace

SsidInfoRow::SsidInfoRow(lv_obj_t *parent)
	: Component(parent)
{
}

SsidInfoRow::~SsidInfoRow()
{
	destroy();
}

void SsidInfoRow::create()
{
	if (_obj != nullptr) {
		return;
	}

	_obj = lv_obj_create(_parent);
	if (_obj == nullptr) {
		return;
	}
	lv_obj_set_size(_obj, WIDTH, HEIGHT);
	lv_obj_set_style_bg_opa(_obj, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_radius(_obj, SConA::RADIUS, LV_PART_MAIN);
	lv_obj_set_style_border_width(_obj, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(_obj, 0, LV_PART_MAIN);
	lv_obj_set_style_clip_corner(_obj, true, LV_PART_MAIN);
	lv_obj_clear_flag(_obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_obj, LV_OBJ_FLAG_SCROLLABLE);

	_key_label = lv_label_create(_obj);
	if (_key_label != nullptr) {
		lv_label_set_text(_key_label, "SSID:");
		lv_obj_set_style_text_color(_key_label, lv_color_white(), LV_PART_MAIN);
		lv_obj_set_style_text_font(_key_label, &lv_font_montserrat_20,
					   LV_PART_MAIN);
		lv_obj_align(_key_label, LV_ALIGN_LEFT_MID, KEY_HORIZONTAL_INSET, 0);
		configure_noninteractive(_key_label);
	}

	_fallback_label = lv_label_create(_obj);
	if (_fallback_label != nullptr) {
		const lv_font_t *const fallback_font = compiled_wifi_ssid_font();
		const int32_t font_height = lv_font_get_line_height(fallback_font);
		lv_obj_set_pos(_fallback_label, VALUE_X,
			       (static_cast<int32_t>(HEIGHT) - font_height) / 2);
		lv_obj_set_size(_fallback_label, VALUE_WIDTH, font_height);
		lv_obj_set_style_bg_opa(_fallback_label, LV_OPA_TRANSP, LV_PART_MAIN);
		lv_obj_set_style_border_width(_fallback_label, 0, LV_PART_MAIN);
		lv_obj_set_style_pad_all(_fallback_label, 0, LV_PART_MAIN);
		lv_obj_set_style_text_color(_fallback_label, lv_color_white(),
					    LV_PART_MAIN);
		lv_obj_set_style_text_font(_fallback_label, fallback_font,
					   LV_PART_MAIN);
		lv_obj_set_style_text_align(_fallback_label, LV_TEXT_ALIGN_RIGHT,
					     LV_PART_MAIN);
		lv_label_set_text(_fallback_label, _fallback_text.c_str());
		configure_noninteractive(_fallback_label);
		apply_fallback_scroll_mode();
	}

	const std::size_t stride = static_cast<std::size_t>(
		LV_DRAW_BUF_STRIDE(VALUE_WIDTH, LV_COLOR_FORMAT_ARGB8888));
	const std::size_t buffer_bytes = stride * HEIGHT;
	if (stride == 0 || buffer_bytes / stride != HEIGHT) {
		spdlog::warn("[wifi] SSID information canvas geometry is invalid");
		return;
	}

	try {
		_canvas_storage.assign(buffer_bytes, 0U);
	} catch (const std::bad_alloc &) {
		spdlog::warn("[wifi] SSID information canvas allocation failed: bytes={}",
			     buffer_bytes);
		return;
	}

	_value_canvas = lv_canvas_create(_obj);
	if (_value_canvas == nullptr) {
		_canvas_storage.clear();
		spdlog::warn("[wifi] SSID information canvas creation failed");
		return;
	}
	lv_canvas_set_buffer(_value_canvas, _canvas_storage.data(), VALUE_WIDTH,
			     HEIGHT, LV_COLOR_FORMAT_ARGB8888);
	lv_obj_set_pos(_value_canvas, VALUE_X, 0);
	lv_obj_set_size(_value_canvas, VALUE_WIDTH, HEIGHT);
	lv_obj_set_style_bg_opa(_value_canvas, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_border_width(_value_canvas, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(_value_canvas, 0, LV_PART_MAIN);
	lv_obj_set_style_radius(_value_canvas, 0, LV_PART_MAIN);
	configure_noninteractive(_value_canvas);
	lv_obj_add_flag(_value_canvas, LV_OBJ_FLAG_HIDDEN);
	clear_canvas();
}

void SsidInfoRow::destroy()
{
	_raster_marquee.reset();
	stop_fallback_scroll();
	clear_canvas();
	_key_label = nullptr;
	_value_canvas = nullptr;
	_fallback_label = nullptr;
	Component::destroy();
	_canvas_storage.clear();
	_image_rendered = false;
}

bool SsidInfoRow::set_image_source(const lv_draw_buf_t *source)
{
	if (!source_is_compatible(source) || !canvas_available()) {
		return false;
	}
	clear_raster_marquee();

	clear_canvas();
	lv_layer_t layer {};
	lv_canvas_init_layer(_value_canvas, &layer);
	if (layer.draw_buf == nullptr) {
		return false;
	}

	lv_draw_image_dsc_t draw_dsc;
	lv_draw_image_dsc_init(&draw_dsc);
	draw_dsc.src = source;
	lv_draw_image(&layer, &draw_dsc, &VALUE_DRAW_AREA);
	lv_canvas_finish_layer(_value_canvas, &layer);
	_image_rendered = true;
	lv_obj_remove_flag(_value_canvas, LV_OBJ_FLAG_HIDDEN);
	if (_fallback_label != nullptr && lv_obj_is_valid(_fallback_label)) {
		stop_fallback_scroll();
		lv_obj_add_flag(_fallback_label, LV_OBJ_FLAG_HIDDEN);
	}
	return true;
}

bool SsidInfoRow::set_prepared_raster(const SsidPreparedRaster &raster,
				      bool start_scroll)
{
	// Install and copy the worker-owned pixels immediately, even while the page
	// is hidden. Entering the page must only reveal an already prepared source.
	return install_prepared_raster_now(raster, start_scroll);
}

bool SsidInfoRow::install_prepared_raster_now(
	const SsidPreparedRaster &raster, bool start_scroll)
{
	if (_obj == nullptr || !lv_obj_is_valid(_obj) || !raster.has_valid_storage() ||
	    raster.source_height != HEIGHT) {
		return false;
	}

	if (raster.overflows_viewport) {
		if (raster.source_width <= VALUE_WIDTH ||
		    raster.text_width <= static_cast<int32_t>(VALUE_WIDTH)) {
			return false;
		}
		if (_raster_marquee == nullptr) {
			try {
				_raster_marquee = std::make_unique<SsidRasterMarquee>(
					_obj, VALUE_WIDTH, HEIGHT, nullptr, 0);
			} catch (const std::bad_alloc &) {
				return false;
			}
			if (_raster_marquee->get_obj() == nullptr) {
				clear_raster_marquee();
				return false;
			}
			lv_obj_set_pos(_raster_marquee->get_obj(), VALUE_X, 0);
			lv_obj_move_to_index(_raster_marquee->get_obj(), -1);
		}
		if (!_raster_marquee->install_prepared_raster(raster,
						       start_scroll && !_fallback_scroll_paused)) {
			return false;
		}
		if (_value_canvas != nullptr && lv_obj_is_valid(_value_canvas)) {
			lv_obj_add_flag(_value_canvas, LV_OBJ_FLAG_HIDDEN);
		}
		if (_fallback_label != nullptr && lv_obj_is_valid(_fallback_label)) {
			stop_fallback_scroll();
			lv_obj_add_flag(_fallback_label, LV_OBJ_FLAG_HIDDEN);
		}
		return true;
	}

	if (raster.source_width != VALUE_WIDTH) {
		return false;
	}
	const std::size_t stride = static_cast<std::size_t>(
		LV_DRAW_BUF_STRIDE(VALUE_WIDTH, LV_COLOR_FORMAT_ARGB8888));
	const std::size_t byte_count = stride * HEIGHT;
	if (stride == 0 || byte_count / stride != HEIGHT ||
	    byte_count > raster.identity.byte_budget) {
		return false;
	}

	try {
		std::vector<uint8_t> source_storage(byte_count, 0U);
		lv_draw_buf_t source {};
		if (lv_draw_buf_init(&source, VALUE_WIDTH, HEIGHT,
				     LV_COLOR_FORMAT_ARGB8888, LV_STRIDE_AUTO,
				     source_storage.data(),
				     static_cast<uint32_t>(source_storage.size())) != LV_RESULT_OK) {
			return false;
		}
		for (uint16_t y = 0; y < HEIGHT; ++y) {
			uint8_t *const destination = source_storage.data() +
				static_cast<std::size_t>(y) * source.header.stride;
			for (uint16_t x = 0; x < VALUE_WIDTH; ++x) {
				const uint32_t pixel = (*raster.pixels)[
					static_cast<std::size_t>(y) * VALUE_WIDTH + x];
				destination[static_cast<std::size_t>(x) * 4U] =
					static_cast<uint8_t>(pixel);
				destination[static_cast<std::size_t>(x) * 4U + 1U] =
					static_cast<uint8_t>(pixel >> 8U);
				destination[static_cast<std::size_t>(x) * 4U + 2U] =
					static_cast<uint8_t>(pixel >> 16U);
				destination[static_cast<std::size_t>(x) * 4U + 3U] =
					static_cast<uint8_t>(pixel >> 24U);
			}
		}
		return set_image_source(&source);
	} catch (const std::bad_alloc &) {
		return false;
	}
}

bool SsidInfoRow::set_fallback_text(const char *text)
{
	if (_fallback_label == nullptr || !lv_obj_is_valid(_fallback_label)) {
		return false;
	}
	clear_raster_marquee();

	const std::string fallback_text = text != nullptr ? text : "";
	if (_image_rendered) {
		clear_canvas();
	}
	if (_value_canvas != nullptr && lv_obj_is_valid(_value_canvas)) {
		lv_obj_add_flag(_value_canvas, LV_OBJ_FLAG_HIDDEN);
	}
	if (_fallback_text != fallback_text) {
		_fallback_text = fallback_text;
		lv_label_set_text(_fallback_label, _fallback_text.c_str());
	}
	lv_obj_remove_flag(_fallback_label, LV_OBJ_FLAG_HIDDEN);
	apply_fallback_scroll_mode();
	return true;
}

void SsidInfoRow::set_fallback_scroll_enabled(bool enabled)
{
	_fallback_scroll_enabled = enabled;
	apply_fallback_scroll_mode();
}

void SsidInfoRow::clear_raster_marquee() noexcept
{
	_raster_marquee.reset();
	if (_value_canvas != nullptr && lv_obj_is_valid(_value_canvas)) {
		if (_image_rendered) {
			lv_obj_remove_flag(_value_canvas, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_add_flag(_value_canvas, LV_OBJ_FLAG_HIDDEN);
		}
	}
	if (_fallback_label != nullptr && lv_obj_is_valid(_fallback_label)) {
		if (_image_rendered) {
			stop_fallback_scroll();
			lv_obj_add_flag(_fallback_label, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_remove_flag(_fallback_label, LV_OBJ_FLAG_HIDDEN);
			apply_fallback_scroll_mode();
		}
	}
}

void SsidInfoRow::pause_ssid_scroll() noexcept
{
	_fallback_scroll_paused = true;
	stop_fallback_scroll();
	if (_raster_marquee != nullptr) {
		_raster_marquee->pause();
	}
}

void SsidInfoRow::resume_ssid_scroll() noexcept
{
	_fallback_scroll_paused = false;
	if (_raster_marquee != nullptr) {
		_raster_marquee->resume();
	}
	apply_fallback_scroll_mode();
}

bool SsidInfoRow::canvas_available() const noexcept
{
	return _value_canvas != nullptr && lv_obj_is_valid(_value_canvas) &&
	       !_canvas_storage.empty();
}

bool SsidInfoRow::source_is_compatible(
	const lv_draw_buf_t *source) const noexcept
{
	if (source == nullptr || source->data == nullptr ||
	    source->header.w != VALUE_WIDTH || source->header.h != HEIGHT ||
	    source->header.cf != LV_COLOR_FORMAT_ARGB8888 ||
	    source->header.stride == 0) {
		return false;
	}

	const std::size_t stride = source->header.stride;
	const std::size_t required_stride = static_cast<std::size_t>(
		LV_DRAW_BUF_STRIDE(VALUE_WIDTH, LV_COLOR_FORMAT_ARGB8888));
	return stride >= required_stride && stride <= source->data_size / HEIGHT;
}

void SsidInfoRow::clear_canvas()
{
	if (!canvas_available()) {
		_image_rendered = false;
		return;
	}
	lv_canvas_fill_bg(_value_canvas, lv_color_black(), LV_OPA_TRANSP);
	_image_rendered = false;
}

void SsidInfoRow::apply_fallback_scroll_mode() noexcept
{
	if (_fallback_label == nullptr || !lv_obj_is_valid(_fallback_label)) {
		return;
	}
	const bool should_scroll = _fallback_scroll_enabled &&
		!_fallback_scroll_paused &&
		!lv_obj_has_flag(_fallback_label, LV_OBJ_FLAG_HIDDEN);
	lv_label_set_long_mode(_fallback_label, should_scroll ?
		LV_LABEL_LONG_SCROLL_CIRCULAR : LV_LABEL_LONG_CLIP);
}

void SsidInfoRow::stop_fallback_scroll() noexcept
{
	if (_fallback_label != nullptr && lv_obj_is_valid(_fallback_label)) {
		lv_label_set_long_mode(_fallback_label, LV_LABEL_LONG_CLIP);
	}
}

} // namespace ui
