#include <algorithm>
#include <csignal>
#include <vector>

#include <tools/state_manager.hpp>
#include <tools/termination_handler.hpp>

namespace tools {

static const std::vector<decltype(SIGTERM)> SIGNALS = {SIGTERM, SIGINT};

void TerminatorHandler::handler(int signal)
{
    if (!std::ranges::any_of(SIGNALS, [signal](decltype(signal) sig) {
            return sig == signal;
        })) {
        return;
    }

    [[maybe_unused]] auto _ =
        StateManager::get_instance().set_state("need_exit", "true");
}

void TerminatorHandler::init()
{
    for (auto sig : SIGNALS) {
        signal(sig, handler);
    }
}

}  // namespace tools
