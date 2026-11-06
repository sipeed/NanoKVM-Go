#ifndef __KVM_UI_TOOLS_TERMINATION_HANDLER_HPP__
#define __KVM_UI_TOOLS_TERMINATION_HANDLER_HPP__

namespace tools {

class TerminatorHandler final {
  public:
    static void init();

  private:
    static void handler(int signal);
};

}  // namespace tools

#endif  // __KVM_UI_TOOLS_TERMINATION_HANDLER_HPP__
