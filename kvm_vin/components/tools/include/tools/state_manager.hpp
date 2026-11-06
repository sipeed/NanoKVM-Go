#ifndef __KVM_UI_STATE_MANAGER_HPP__
#define __KVM_UI_STATE_MANAGER_HPP__

#include <memory>
#include <optional>
#include <shared_mutex>
#include <string>
#include <string_view>

namespace tools {

class StateManager final {
  public:
    enum class Error {
        succ = 0,
        not_found,
        concurrency_conflict,
        invalid_argument
    };

  public:
    [[nodiscard]] static StateManager& get_instance() noexcept;

    StateManager(const StateManager&) = delete;
    StateManager& operator=(const StateManager&) = delete;
    StateManager(StateManager&&) = delete;
    StateManager& operator=(StateManager&&) = delete;

    Error set_state(std::string_view key,
                    std::string_view value) noexcept;

    std::optional<std::string> get_state(
        std::string_view key) const noexcept;

  private:
    struct Impl;

  private:
    StateManager();
    ~StateManager() = default;

  private:
    mutable std::shared_mutex _mutex;
    std::unique_ptr<Impl> _impl;
};

}  // namespace tools

#endif  // __KVM_UI_STATE_MANAGER_HPP__
