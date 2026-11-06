#include <fstream>

// logger
#include <logger/default2.hpp>
#include <logger/file_sink.hpp>
#include <logger/tools.hpp>

#include "lt7911_utils.hpp"

/* Logger */
void Lt7911Utils::logger_init(const std::string& level,
                                   const std::string& type,
                                   const std::string& ofilepath)
{
    /* Use Default Logger */
    this->_logger = std::make_shared<DefaultLogger2>("Lt7911Utils");
    this->_logger->logger_init(level, type, ofilepath);
}

std::shared_ptr<spdlog::logger> Lt7911Utils::operator->()
{
    return this->_logger->operator->();
}

Lt7911Utils::Lt7911Utils(const std::string& proc_base_path) : _base_path(proc_base_path)
{
    if (_base_path.empty()) {
        _base_path = "/proc/lt7911_info";
    }
}

bool Lt7911Utils::_read_int_from_file(const std::string& filename, int& value) {
    std::ifstream file(_base_path + "/" + filename);
    if (!(file >> value)) {
        value = 0;
        return false;
    }
    return true;
}

bool Lt7911Utils::read_status(std::string& status) {
    std::ifstream file(_base_path + "/status");
    if (!file || !std::getline(file, status)) {
        return false;
    }
    return true;
}

bool Lt7911Utils::write_status(const std::string& status) {
    const std::string path = _base_path + "/status";
    std::ofstream file(path);
    if (!file) {
        (*_logger)->error("Failed to open file for writing: {}", path);
        return false;
    }
    file << status << std::endl;
    if (!file) {
        (*_logger)->error("Failed to write status to file: {}", path);
        return false;
    }
    return true;
}

bool Lt7911Utils::reset_lt7911(bool enable) {
    const std::string path = _base_path + "/power";
    std::ofstream file(path);
    if (!file) {
        (*_logger)->error("Failed to open Lt7911 power control file: {}", path);
        return false;
    }

    file << (enable ? "1" : "0") << std::endl;
    if (!file.good()) {
        (*_logger)->error("Failed to write Lt7911 power value: {}", enable);
        return false;
    }

    (*_logger)->info("Lt7911 power {}abled", enable ? "en" : "dis");
    return true;
}

bool Lt7911Utils::read_res(int& width, int& height, int& fps) {
    bool ok = true;
    ok &= _read_int_from_file("width", width);
    ok &= _read_int_from_file("height", height);
    ok &= _read_int_from_file("fps", fps);
    return ok && width > 0 && height > 0 && fps > 0;
}
