#include <cstddef>
#include <memory>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_test_context.hpp"
#include "ui/servo_page.hpp"

namespace ui
{

class ServoPageTestAccess final {
    public:
	static void set_duties(ServoPage &page, int idle_ns, int zero_ns)
	{
		page._idle_ns = idle_ns;
		page._zero_ns = zero_ns;
		page.update_slider_visual();
		page.update_title();
		page.update_zero_label();
	}

	static void set_idle(ServoPage &page, int idle_ns)
	{
		page.set_idle_ns(idle_ns, true);
	}

	static lv_obj_t *click_button(const ServoPage &page)
	{
		return page._click_btn == nullptr ? nullptr :
						    page._click_btn->get_obj();
	}

	static lv_obj_t *free_button(const ServoPage &page)
	{
		return page._free_btn == nullptr ? nullptr :
						   page._free_btn->get_obj();
	}

	static bool has_active_click_sequence(const ServoPage &page)
	{
		return page.has_active_click_sequence();
	}

	static bool has_click_timer(const ServoPage &page)
	{
		return page._click_timer != nullptr;
	}
};

} // namespace ui

namespace
{

constexpr int kIdleNs = 1600000;
constexpr int kZeroNs = 1200000;

class RecordingPwmBackend final : public ui::servo_pwm::Backend {
    public:
	ui::servo_pwm::ApplyResult initialize(int idle_ns) override
	{
		_initialize_values.push_back(idle_ns);
		_last_successful_duty_ns = idle_ns;
		return { .ok = true };
	}

	ui::servo_pwm::ApplyResult write_idle(int idle_ns) override
	{
		_write_values.push_back(idle_ns);
		if (_fail_writes) {
			return { .error = "injected write failure" };
		}
		_last_successful_duty_ns = idle_ns;
		return { .ok = true };
	}

	void set_write_failure(bool fail)
	{
		_fail_writes = fail;
	}

	int last_successful_duty_ns() const
	{
		return _last_successful_duty_ns;
	}

	const std::vector<int> &initialize_values() const
	{
		return _initialize_values;
	}

	const std::vector<int> &write_values() const
	{
		return _write_values;
	}

    private:
	bool _fail_writes = false;
	int _last_successful_duty_ns = -1;
	std::vector<int> _initialize_values;
	std::vector<int> _write_values;
};

class ServoPageFixture final {
    public:
	ServoPageFixture()
		: _page(make_backend(&_backend))
	{
		_page.create(_lvgl.screen());
		ui::ServoPageTestAccess::set_duties(_page, kIdleNs, kZeroNs);
		lv_obj_update_layout(_lvgl.screen());
		_initial_timer_count = _lvgl.timer_count();
	}

	~ServoPageFixture()
	{
		if (!_destroyed) {
			_page.destroy();
		}
	}

	ServoPageFixture(const ServoPageFixture &) = delete;
	ServoPageFixture &operator=(const ServoPageFixture &) = delete;

	test_support::LvglTestContext &lvgl()
	{
		return _lvgl;
	}

	ui::ServoPage &page()
	{
		return _page;
	}

	int duty_ns() const
	{
		return _backend->last_successful_duty_ns();
	}

	void set_write_failure(bool fail)
	{
		_backend->set_write_failure(fail);
	}

	const std::vector<int> &initialize_values() const
	{
		return _backend->initialize_values();
	}

	const std::vector<int> &write_values() const
	{
		return _backend->write_values();
	}

	void set_idle_ns(int idle_ns)
	{
		ui::ServoPageTestAccess::set_idle(_page, idle_ns);
	}

	std::size_t initial_timer_count() const
	{
		return _initial_timer_count;
	}

	void start_click()
	{
		lv_obj_t *button = ui::ServoPageTestAccess::click_button(_page);
		REQUIRE(button != nullptr);
		lv_obj_send_event(button, LV_EVENT_CLICKED, nullptr);
	}

	void enable_free_mode()
	{
		lv_obj_t *button = ui::ServoPageTestAccess::free_button(_page);
		REQUIRE(button != nullptr);
		lv_obj_send_event(button, LV_EVENT_CLICKED, nullptr);
	}

	bool has_active_click_sequence() const
	{
		return ui::ServoPageTestAccess::has_active_click_sequence(
			_page);
	}

	bool has_click_timer() const
	{
		return ui::ServoPageTestAccess::has_click_timer(_page);
	}

	void destroy_page()
	{
		_page.destroy();
		_destroyed = true;
	}

    private:
	static std::unique_ptr<ui::servo_pwm::Backend>
	make_backend(RecordingPwmBackend **backend)
	{
		auto recording = std::make_unique<RecordingPwmBackend>();
		*backend = recording.get();
		return recording;
	}

	test_support::LvglTestContext _lvgl;
	RecordingPwmBackend *_backend = nullptr;
	ui::ServoPage _page;
	std::size_t _initial_timer_count = 0;
	bool _destroyed = false;
};

std::size_t require_active_click(ServoPageFixture &fixture)
{
	fixture.start_click();
	REQUIRE(fixture.has_active_click_sequence());
	REQUIRE(fixture.has_click_timer());
	const std::size_t timer_count = fixture.lvgl().timer_count();
	CHECK(timer_count > fixture.initial_timer_count());
	CHECK(fixture.duty_ns() == kZeroNs);
	return timer_count;
}

void require_click_cancelled(ServoPageFixture &fixture,
			     std::size_t active_timer_count)
{
	CHECK_FALSE(fixture.has_active_click_sequence());
	CHECK_FALSE(fixture.has_click_timer());
	CHECK(fixture.lvgl().timer_count() < active_timer_count);
}

void require_no_later_click_write(ServoPageFixture &fixture)
{
	fixture.lvgl().advance(2 * ui::servo_pwm::kClickStepDelayMs + 20);
	CHECK_FALSE(fixture.has_active_click_sequence());
	CHECK_FALSE(fixture.has_click_timer());
	CHECK(fixture.duty_ns() == kZeroNs);
}

} // namespace

TEST_CASE(
	"Servo page cancels interrupted click sequences at every lifecycle boundary",
	"[ui][servo]")
{
	SECTION("pause after the initial zero step")
	{
		ServoPageFixture fixture;
		const std::size_t active_timer_count =
			require_active_click(fixture);

		fixture.page().on_pause();
		require_click_cancelled(fixture, active_timer_count);
		CHECK(fixture.duty_ns() == kZeroNs);
		fixture.page().on_exit();
		require_no_later_click_write(fixture);
	}

	SECTION("exit after the middle idle step")
	{
		ServoPageFixture fixture;
		require_active_click(fixture);
		fixture.lvgl().advance(ui::servo_pwm::kClickStepDelayMs);
		REQUIRE(fixture.has_active_click_sequence());
		CHECK(fixture.duty_ns() == kIdleNs);
		const std::size_t active_timer_count =
			fixture.lvgl().timer_count();

		fixture.page().on_exit();
		require_click_cancelled(fixture, active_timer_count);
		CHECK(fixture.duty_ns() == kZeroNs);
		require_no_later_click_write(fixture);
	}

	SECTION("destroy after the middle idle step")
	{
		ServoPageFixture fixture;
		require_active_click(fixture);
		fixture.lvgl().advance(ui::servo_pwm::kClickStepDelayMs);
		REQUIRE(fixture.has_active_click_sequence());
		CHECK(fixture.duty_ns() == kIdleNs);
		const std::size_t active_timer_count =
			fixture.lvgl().timer_count();

		fixture.destroy_page();
		require_click_cancelled(fixture, active_timer_count);
		CHECK(fixture.duty_ns() == kZeroNs);
		require_no_later_click_write(fixture);
	}
}

TEST_CASE("Servo Free mode retains its output on page exit", "[ui][servo]")
{
	ServoPageFixture fixture;
	fixture.enable_free_mode();
	REQUIRE_FALSE(fixture.has_active_click_sequence());
	REQUIRE(fixture.duty_ns() == kIdleNs);

	fixture.page().on_pause();
	fixture.page().on_exit();
	CHECK(fixture.duty_ns() == kIdleNs);
}

TEST_CASE(
	"Servo page applies Free and Click duties through its injected backend",
	"[ui][servo]")
{
	SECTION("Free initialization and subsequent slider change")
	{
		ServoPageFixture fixture;
		fixture.enable_free_mode();
		REQUIRE(fixture.initialize_values().size() == 1U);
		CHECK(fixture.initialize_values()[0] == kIdleNs);
		CHECK(fixture.write_values().empty());

		fixture.set_idle_ns(1700000);
		REQUIRE(fixture.write_values().size() == 1U);
		CHECK(fixture.write_values()[0] == 1700000);
	}

	SECTION("Click emits zero, selected duty, then zero")
	{
		ServoPageFixture fixture;
		fixture.start_click();
		fixture.lvgl().advance(ui::servo_pwm::kClickStepDelayMs);
		fixture.lvgl().advance(ui::servo_pwm::kClickStepDelayMs);

		REQUIRE(fixture.initialize_values().size() == 1U);
		CHECK(fixture.initialize_values()[0] == kZeroNs);
		REQUIRE(fixture.write_values().size() == 3U);
		CHECK(fixture.write_values()[0] == kZeroNs);
		CHECK(fixture.write_values()[1] == kIdleNs);
		CHECK(fixture.write_values()[2] == kZeroNs);
	}
}

TEST_CASE("Servo interruption remains cancelled when the zero write fails",
	  "[ui][servo]")
{
	ServoPageFixture fixture;
	const std::size_t active_timer_count = require_active_click(fixture);
	const std::size_t write_count_before_failure =
		fixture.write_values().size();
	fixture.set_write_failure(true);

	fixture.page().on_pause();
	require_click_cancelled(fixture, active_timer_count);
	CHECK(fixture.write_values().size() == write_count_before_failure + 1U);
	fixture.lvgl().advance(2 * ui::servo_pwm::kClickStepDelayMs + 20);
	CHECK(fixture.write_values().size() == write_count_before_failure + 1U);
	CHECK_FALSE(fixture.has_active_click_sequence());
	CHECK_FALSE(fixture.has_click_timer());
}
