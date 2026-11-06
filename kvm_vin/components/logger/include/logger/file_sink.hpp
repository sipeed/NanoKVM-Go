#ifndef __KVM_UI_LOGGERS_FILE_SINK_HPP__
#define __KVM_UI_LOGGERS_FILE_SINK_HPP__

#include <memory>
#include <shared_mutex>
#include <string>
#include <string_view>

#include <spdlog/sinks/rotating_file_sink.h>

class RotatingSink final {
  public:
    using Sink = std::shared_ptr<spdlog::sinks::rotating_file_sink_mt>;

    static constexpr std::string_view DEF_PATH = "/var/log/kvmcomm/kvm_ui_.log";
    static constexpr size_t DEF_SIZE = 2 * 1024 * 1024;
    static constexpr size_t DEF_FILES = 3;

    static void configure(std::string_view path, size_t size, size_t files);

    static Sink get_instance();

  private:
    static Sink create_sink();

  private:
    inline static std::shared_mutex _mutex;
    inline static bool _init {false};
    inline static std::string _file_path;
    inline static size_t _max_size;
    inline static size_t _max_files;
};

#endif  // __KVM_UI_LOGGERS_FILE_SINK_HPP__
