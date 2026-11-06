#pragma once

#include <cstddef>
#include <vector>

#include "ui/page.hpp"

namespace ui
{

class SubPageStack {
    public:
	void clear();
	void push(Page *page);
	Page *pop();

	Page *top() const;
	std::size_t size() const
	{
		return _pages.size();
	}
	bool empty() const
	{
		return _pages.empty();
	}

    private:
	std::vector<Page *> _pages;
};

} // namespace ui
