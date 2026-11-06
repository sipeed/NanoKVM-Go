#include "ui/subpage_stack.hpp"

#include <algorithm>
#include <iterator>

namespace ui
{

namespace
{

void hide_page(Page *page)
{
	if (page != nullptr && page->is_created()) {
		lv_obj_add_flag(page->get_container(), LV_OBJ_FLAG_HIDDEN);
	}
}

void show_page(Page *page)
{
	if (page != nullptr && page->is_created()) {
		lv_obj_clear_flag(page->get_container(), LV_OBJ_FLAG_HIDDEN);
	}
}

} // namespace

void SubPageStack::clear()
{
	while (!_pages.empty()) {
		Page *page = _pages.back();
		_pages.pop_back();
		if (page != nullptr) {
			page->on_exit();
		}
		hide_page(page);
	}
}

void SubPageStack::push(Page *page)
{
	if (page == nullptr || !page->is_created()) {
		return;
	}

	const auto existing = std::find(_pages.begin(), _pages.end(), page);
	if (existing != _pages.end()) {
		const auto target_index = static_cast<std::size_t>(
			std::distance(_pages.begin(), existing));
		if (target_index + 1U == _pages.size()) {
			show_page(page);
			return;
		}
		while (_pages.size() > target_index + 1U) {
			Page *top_page = _pages.back();
			_pages.pop_back();
			if (top_page != nullptr) {
				top_page->on_exit();
			}
			hide_page(top_page);
		}
		show_page(page);
		page->on_resume();
		return;
	}

	if (!_pages.empty()) {
		Page *current = _pages.back();
		if (current != nullptr) {
			current->on_pause();
		}
		hide_page(current);
	}

	_pages.push_back(page);
	show_page(page);
	page->on_enter();
}

Page *SubPageStack::pop()
{
	if (_pages.empty()) {
		return nullptr;
	}

	Page *current = _pages.back();
	_pages.pop_back();
	if (current != nullptr) {
		current->on_exit();
	}
	hide_page(current);

	Page *restored = top();
	if (restored != nullptr) {
		show_page(restored);
		restored->on_resume();
	}
	return restored;
}

Page *SubPageStack::top() const
{
	return _pages.empty() ? nullptr : _pages.back();
}

} // namespace ui
