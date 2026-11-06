#pragma once

#include <exception>
#include <functional>
#include <optional>
#include <string>
#include <string_view>

namespace lvgl_main
{

std::optional<std::string> parse_deb_package_version(std::string_view output);

class StartupPackageVersionCache final {
    public:
	using Loader = std::function<std::optional<std::string>()>;

	explicit StartupPackageVersionCache(Loader loader);
	StartupPackageVersionCache(const StartupPackageVersionCache &) = delete;
	StartupPackageVersionCache &
	operator=(const StartupPackageVersionCache &) = delete;
	StartupPackageVersionCache(StartupPackageVersionCache &&) = delete;
	StartupPackageVersionCache &
	operator=(StartupPackageVersionCache &&) = delete;

	const std::string &load_once();
	const std::string &value() const noexcept
	{
		return _version;
	}

    private:
	Loader _loader;
	std::string _version;
	std::exception_ptr _error;
	bool _loaded { false };
};

} // namespace lvgl_main
