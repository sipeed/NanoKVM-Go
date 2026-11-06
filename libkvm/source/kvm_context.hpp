#pragma once

#include <atomic>
#include <memory>
#include <string>
#include "libkvm/kvm_vision.h"
#include "logger_default.hpp"
#include "control_client.hpp"
#include "video_socket/unix_video_frame_reader.hpp"
#include "../../kvm_vin/source/include/video_frame_protocol.h"

struct EncoderState {
    std::atomic<uint16_t> gop{0};
    std::atomic<uint8_t> fps{0};
    std::atomic<uint8_t> rate_control_mode{VENC_CBR};
};

class KVMContext {
public:
    static KVMContext& getInstance() {
        static KVMContext instance;
        return instance;
    }

    KVMContext(const KVMContext&) = delete;
    KVMContext& operator=(const KVMContext&) = delete;

    EncoderState& getEncoderState() { return encoder_state_; }

    UnixVideoFrameReader& getVideoReader() { return video_reader_; }
    KvmVinControlClient& getControlClient() { return control_client_; }

    std::shared_ptr<ILogger>& getLogger() { return logger_; }
    void setLogger(std::shared_ptr<ILogger> logger) { logger_ = logger; }

    std::string& getLoggerLevel() { return logger_level_; }
    void setLoggerLevel(const std::string& level) { logger_level_ = level; }

    bool isDeinitialized() const { return is_deinitialized_.load(); }
    void setDeinitialized(bool flag) { is_deinitialized_.store(flag); }

    bool isHdmiEnabled() const { return hdmi_enabled_.load(); }
    void setHdmiEnabled(bool flag) { hdmi_enabled_.store(flag); }

private:
    KVMContext() {
        // Setup IDR request callback for startup, reconnect, and dropped chains.
        video_reader_.set_idr_request_callback([this]() {
            control_client_.request_idr(true);
        });
    }

    EncoderState encoder_state_;

    // control_client_ must outlive video_reader_: the reader's IDR worker
    // thread captures control_client_.
    // Members destroy in reverse declaration order, so declare it first.
    KvmVinControlClient control_client_{};
    UnixVideoFrameReader video_reader_{KVM_VIN_VIDEO_SOCK_PATH};

    std::shared_ptr<ILogger> logger_{nullptr};
    std::string logger_level_;
    std::atomic<bool> is_deinitialized_{false};
    std::atomic<bool> hdmi_enabled_{true};
};
