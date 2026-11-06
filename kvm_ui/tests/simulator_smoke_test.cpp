#include <cerrno>
#include <chrono>
#include <csignal>
#include <filesystem>
#include <iostream>
#include <thread>

#include <sys/wait.h>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/lvgl_main.hpp"
#include "lvgl_main/simulator_build_config.hpp"

namespace
{

constexpr auto kShutdownDelay = std::chrono::milliseconds { 500 };

std::filesystem::path smoke_config_path()
{
	const auto default_path = std::filesystem::path(
		lvgl_main::simulator_build_config::kDefaultConfigPath);
	return default_path.parent_path() / "simulator-smoke.toml";
}

bool no_child_processes_remain()
{
	errno = 0;
	const pid_t result = ::waitpid(-1, nullptr, WNOHANG);
	return result == -1 && errno == ECHILD;
}

} // namespace

int main()
{
	// Keep an initialization failure from making the helper SIGTERM terminate
	// the test process before UILVGLApp installs its graceful handler.
	std::signal(SIGTERM, SIG_IGN);

	std::thread shutdown_thread([]() {
		std::this_thread::sleep_for(kShutdownDelay);
		std::raise(SIGTERM);
	});

	const int result = lvgl_ui_main(lvgl_main::default_app_config(),
					smoke_config_path());
	shutdown_thread.join();

	if (result != 0) {
		std::cerr << "Simulator UI smoke test returned " << result
			  << "\n";
		return 1;
	}
	if (!no_child_processes_remain()) {
		std::cerr << "Simulator UI smoke test left a child process\n";
		return 1;
	}

	return 0;
}
