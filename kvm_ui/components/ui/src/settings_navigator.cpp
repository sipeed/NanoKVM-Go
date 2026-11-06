#include "ui/settings_navigator.hpp"

#include "ui/page.hpp"
#include "ui/page_registry.hpp"

namespace ui
{

SettingsNavigator::SettingsNavigator(RootPager &root_pager,
				     PageRegistry &page_registry)
	: _root_pager(root_pager)
	, _page_registry(page_registry)
{
}

SettingsNavigator::~SettingsNavigator()
{
	shutdown();
}

void SettingsNavigator::set_route_change_callback(RouteChangeCallback callback)
{
	if (_is_shutting_down) {
		return;
	}
	_on_route_change = std::move(callback);
}

void SettingsNavigator::set_return_navigation_guard(ReturnNavigationGuard guard)
{
	if (_is_shutting_down) {
		return;
	}
	_return_navigation_guard = std::move(guard);
}

void SettingsNavigator::begin_shutdown()
{
	if (_is_shutting_down) {
		return;
	}

	_is_shutting_down = true;
	_on_route_change = nullptr;
	_return_navigation_guard = nullptr;
}

void SettingsNavigator::shutdown()
{
	begin_shutdown();
	clear_stack(false);
}

bool SettingsNavigator::show_root_page(RootPager::ContentPage page)
{
	if (!navigation_available()) {
		return false;
	}

	clear_stack(false);
	_root_pager.leave_settings_subpage();
	_root_pager.transition_to(page);
	notify_top_route();
	return true;
}

bool SettingsNavigator::show(SettingsRoute route)
{
	if (!navigation_available()) {
		return false;
	}

	Page *page = _page_registry.page_for(route);
	if (page == nullptr || !page->is_created()) {
		return false;
	}

	_return_anchor = _root_pager.nearest_content_page();
	if (_return_anchor == RootPager::ContentPage::Main) {
		_return_anchor = RootPager::ContentPage::SettingsHome;
	}
	clear_stack(false);
	_root_pager.enter_settings_subpage();
	_subpage_stack.push(page);
	_routes.push_back(route);
	notify_top_route();
	_root_pager.refresh_navigation_chrome();
	return true;
}

bool SettingsNavigator::push(SettingsRoute route)
{
	if (!navigation_available()) {
		return false;
	}

	Page *page = _page_registry.page_for(route);
	if (page == nullptr || !page->is_created()) {
		return false;
	}

	// A route can deliberately be pushed more than once (for example, returning
	// from a nested Wi-Fi flow to its main page). Keep this value stack in the
	// same shape as SubPageStack so pop() always resumes the matching page.
	_routes.push_back(route);
	_root_pager.enter_settings_subpage();
	_subpage_stack.push(page);
	notify_top_route();
	_root_pager.refresh_navigation_chrome();
	return true;
}

bool SettingsNavigator::pop()
{
	if (!navigation_available()) {
		return false;
	}
	if (_routes.size() <= 1U) {
		return return_to_anchor();
	}

	_subpage_stack.pop();
	_routes.pop_back();
	notify_top_route();
	_root_pager.refresh_navigation_chrome();
	return true;
}

bool SettingsNavigator::return_to_anchor()
{
	if (!navigation_available()) {
		return false;
	}
	if (_return_navigation_guard && !_return_navigation_guard()) {
		return false;
	}

	clear_stack(false);
	_root_pager.leave_settings_subpage();
	_root_pager.transition_to(_return_anchor);
	notify_top_route();
	return true;
}

void SettingsNavigator::clear()
{
	if (_is_shutting_down) {
		return;
	}
	clear_stack(true);
}

bool SettingsNavigator::empty() const noexcept
{
	return _routes.empty();
}

std::size_t SettingsNavigator::size() const noexcept
{
	return _routes.size();
}

std::optional<SettingsRoute> SettingsNavigator::top_route() const noexcept
{
	if (_routes.empty()) {
		return std::nullopt;
	}
	return _routes.back();
}

RootPager::ContentPage SettingsNavigator::return_anchor() const noexcept
{
	return _return_anchor;
}

bool SettingsNavigator::navigation_available() const noexcept
{
	return !_is_shutting_down && _root_pager.is_created() &&
	       !_root_pager.is_flip_animating();
}

void SettingsNavigator::clear_stack(bool notify_route_change)
{
	_subpage_stack.clear();
	_routes.clear();
	if (notify_route_change) {
		notify_top_route();
	}
}

void SettingsNavigator::notify_top_route()
{
	if (_on_route_change) {
		_on_route_change(top_route());
	}
}

} // namespace ui
