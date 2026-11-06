#pragma once

#include <string>

class KvmVinControlClient {
public:
    explicit KvmVinControlClient(const std::string& sock_path = "/run/kvm/vin_ctrl.sock")
        : sock_path_(sock_path) {}

    int set_fps(float src_fps, float dst_fps);
    int set_gop(unsigned int gop);
    int set_bitrate(unsigned int bitrate);
    int set_rc_mode(int mode);
    int request_idr(bool instant = true);

private:
    int request(const std::string& payload);

    std::string sock_path_;
};
