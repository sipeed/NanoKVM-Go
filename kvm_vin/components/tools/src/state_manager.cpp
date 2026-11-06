#include <map>
#include <mutex>

#include <tools/state_manager.hpp>

namespace tools {

struct StateManager::Impl {
    std::map<std::string, std::string> map;
};

StateManager::StateManager()
{
    {
        std::unique_lock lock(this->_mutex);
        this->_impl = std::make_unique<Impl>();
    }
}

StateManager& StateManager::get_instance() noexcept
{
    static StateManager instance;
    return instance;
}

StateManager::Error StateManager::set_state(
    std::string_view key, std::string_view value) noexcept
{
    std::unique_lock lock(this->_mutex);
    auto it = this->_impl->map.find(std::string(key));
    if (it != this->_impl->map.end()) {
        it->second = value;
        return Error::succ;
    }
    this->_impl->map.emplace(key, value);
    return Error::succ;
}

std::optional<std::string> StateManager::get_state(
    std::string_view key) const noexcept
{
    Impl impl;
    {
        std::shared_lock lock(this->_mutex);
        impl = *this->_impl;
    }

    auto it = impl.map.find(std::string(key));
    if (it == impl.map.cend()) {
        return std::nullopt;
    }

    return it->second;
}

}  // namespace tools
