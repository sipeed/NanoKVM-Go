#ifndef __KVM_UI_TOOLS_SCHEDULAR_HPP__
#define __KVM_UI_TOOLS_SCHEDULAR_HPP__

#include <chrono>
#include <functional>
#include <queue>
#include <thread>
#include <vector>

namespace tools {

class IntervalScheduler final {
  public:
    using Func = std::function<void()>;
    struct Job {
        Func func;
        std::chrono::milliseconds interval;
        std::chrono::steady_clock::time_point next;
        bool operator>(const Job& rhs) const noexcept;
    };

    void submit(Func f, std::chrono::milliseconds interval);
    void run();

  private:
    std::priority_queue<Job, std::vector<Job>, std::greater<>> _jobs;
};

}  // namespace tools

#endif  // __KVM_UI_TOOLS_SCHEDULAR_HPP__
