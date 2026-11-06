#include "lvgl_main/package_version.hpp"

#include <utility>

namespace lvgl_main
{

std::optional<std::string> parse_deb_package_version(std::string_view output)
{
	const auto first = output.find_first_not_of(" \t\r\n");
	if (first == std::string_view::npos) {
		return std::nullopt;
	}

	const auto last = output.find_last_not_of(" \t\r\n");
	return std::string(output.substr(first, last - first + 1U));
}

StartupPackageVersionCache::StartupPackageVersionCache(Loader loader)
	: _loader(std::move(loader))
{
}

const std::string &StartupPackageVersionCache::load_once()
{
	if (_loaded) {
		if (_error) {
			std::rethrow_exception(_error);
		}
		return _version;
	}
	_loaded = true;
	Loader loader = std::move(_loader);
	_loader = {};
	if (loader) {
		try {
			if (auto version = loader()) {
				_version = std::move(*version);
			}
		} catch (...) {
			_error = std::current_exception();
			throw;
		}
	}
	return _version;
}

} // namespace lvgl_main
