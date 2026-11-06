#pragma once

#include <atomic>
#include <thread>

#include "ax_utils.hpp"
#include <logger/ilogger.hpp>

constexpr const char* KVM_VIN_CTRL_SOCK_PATH = "/run/kvm/vin_ctrl.sock";

std::thread start_control_server_thread(AxeraUtils& axera_utils,
                                        std::atomic<bool>& running,
                                        ILogger& logger);
