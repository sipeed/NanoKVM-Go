#include <csignal>

#include <cxxabi.h>
#include <fmt/core.h>
#include <libunwind.h>
#include <tools/segfault_stack_tracer.h>

namespace tools {

void SegfaultStackTracker::init() { ::signal(SIGSEGV, handler); }

void SegfaultStackTracker::handler(int signal)
{
    if (signal != SIGSEGV) {
        return;
    }
    unw_cursor_t cursor;
    unw_context_t context;

    unw_getcontext(&context);
    unw_init_local(&cursor, &context);

    fmt::println("Stack trace:");
    while (unw_step(&cursor) > 0) {
        unw_word_t offset, pc;
        char sym[256];

        unw_get_reg(&cursor, UNW_REG_IP, &pc);
        if (pc == 0) {
            break;
        }

        char* name = sym;
        if (unw_get_proc_name(&cursor, sym, sizeof(sym), &offset) == 0) {
            int status;
            char* demangled =
                abi::__cxa_demangle(sym, nullptr, nullptr, &status);
            if (demangled) {
                name = demangled;
            }
            fmt::println("0x{:x}: {} + 0x{:x}", pc, name, offset);
            if (demangled) {
                free(demangled);
            }
        } else {
            fmt::println("0x{:x}: ???", pc);
        }
    }
    exit(-1);
}

}  // namespace tools
