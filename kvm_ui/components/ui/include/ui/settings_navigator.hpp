#pragma once

#include <cstddef>
#include <functional>
#include <optional>
#include <vector>

#include "ui/root_pager.hpp"
#include "ui/settings_route.hpp"
#include "ui/subpage_stack.hpp"

namespace ui
{

class PageRegistry;

class SettingsNavigator final {
    public:
	using RouteChangeCallback =
		std::function<void(std::optional<SettingsRoute>)>;
	using ReturnNavigationGuard = std::function<bool()>;

	SettingsNavigator(RootPager &root_pager, PageRegistry &page_registry);
	~SettingsNavigator();

	SettingsNavigator(const SettingsNavigator &) = delete;
	SettingsNavigator &operator=(const SettingsNavigator &) = delete;

	void set_route_change_callback(RouteChangeCallback callback);
	void set_return_navigation_guard(ReturnNavigationGuard guard);
	void begin_shutdown();
	void shutdown();

	bool show_root_page(RootPager::ContentPage page);
	bool show(SettingsRoute route);
	bool push(SettingsRoute route);
	bool pop();
	bool return_to_anchor();
	void clear();

	bool empty() const noexcept;
	std::size_t size() const noexcept;
	std::optional<SettingsRoute> top_route() const noexcept;
	RootPager::ContentPage return_anchor() const noexcept;

    private:
	RootPager &_root_pager;
	PageRegistry &_page_registry;
	SubPageStack _subpage_stack;
	std::vector<SettingsRoute> _routes;
	RootPager::ContentPage _return_anchor =
		RootPager::ContentPage::SettingsHome;
	bool _is_shutting_down = false;
	RouteChangeCallback _on_route_change;
	ReturnNavigationGuard _return_navigation_guard;

	bool navigation_available() const noexcept;
	void clear_stack(bool notify_route_change);
	void notify_top_route();
};

} // namespace ui
