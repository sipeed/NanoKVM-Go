#include <chrono>
#include <thread>

#include <tools/scheduler.hpp>
#include <tools/state_manager.hpp>

namespace tools {

bool IntervalScheduler::Job::operator>(const Job& rhs) const noexcept
{
    return this->next > rhs.next;
}

void IntervalScheduler::submit(Func f, std::chrono::milliseconds interval)
{
    this->_jobs.push(Job {
        std::move(f),
        interval,
        std::chrono::steady_clock::now() + interval,
    });
}

void IntervalScheduler::run()
{
    auto& state_manager = StateManager::get_instance();
    while (state_manager.get_state("need_exit").value_or("") != "true"
           && !this->_jobs.empty()) {
        auto now = std::chrono::steady_clock::now();

        auto job = this->_jobs.top();
        this->_jobs.pop();

        if (job.next > now) {
            std::this_thread::sleep_for(job.next - now);
        }

        job.func();

        job.next += job.interval;
        this->_jobs.push(std::move(job));
    }
}

}  // namespace tools
