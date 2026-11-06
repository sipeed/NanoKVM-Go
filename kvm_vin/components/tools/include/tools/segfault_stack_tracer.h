#ifndef __KVM_SEGFAULT_STACK_TRACKER_HPP__
#define __KVM_SEGFAULT_STACK_TRACKER_HPP__

namespace tools {

class SegfaultStackTracker final {
  public:
    static void init();

  private:
    static void handler(int signal);
};

}  // namespace tools

#endif  // __KVM_SEGFAULT_STACK_TRACKER_HPP__
