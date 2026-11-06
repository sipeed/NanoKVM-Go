#include <logger/file_sink.hpp>

void RotatingSink::configure(std::string_view path, size_t size, size_t files)
{
    std::unique_lock lock(_mutex);
    _file_path = path;
    _max_size = size;
    _max_files = files;
    _init = true;
}

RotatingSink::Sink RotatingSink::create_sink()
{
    std::shared_lock lock(_mutex);
    return std::make_shared<spdlog::sinks::rotating_file_sink_mt>(
        _file_path, _max_size, _max_files);
}

RotatingSink::Sink RotatingSink::get_instance()
{
    static const auto sink = create_sink();
    return sink;
}
