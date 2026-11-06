#ifndef OCR_LOG_H
#define OCR_LOG_H

#include <chrono>
#include <cstdarg>
#include <string>
#include <vector>

// 日志等级
enum class LogLevel {
    NONE = 0,  // 不打印任何日志
    ERROR = 1,  // 只打印错误日志
    INFO = 2  // 打印所有日志（INFO + ERROR）
};

// 初始化日志系统
void log_init(LogLevel level);

// 设置日志等级
void log_set_level(LogLevel level);

// 获取当前日志等级
LogLevel log_get_level();

// 日志函数
void log(LogLevel level,
         const std::string& tag,
         const char* format,
         ...);

// 便捷函数
void log_info(const std::string& tag, const char* format, ...);
void log_warn(const std::string& tag, const char* format, ...);
void log_error(const std::string& tag, const char* format, ...);

// 将字符串转换为日志等级
LogLevel string_to_log_level(const std::string& str);

// 将日志等级转换为字符串
std::string log_level_to_string(LogLevel level);

// ============================================
// 性能分析工具（计时和内存监控）
// ============================================

// 内存信息结构体
struct MemoryInfo {
    long vmRSS;  // 实际使用的物理内存
    long vmPeak;  // 峰值内存
    long vmSize;  // 虚拟内存
};

// 获取当前进程内存使用信息
MemoryInfo get_memory_info();

// 打印内存使用信息
void log_memory(const std::string& step);

// 打印最终内存使用总结
void log_final_memory_summary();

// 时间戳管理（最多支持 50 个步骤）
#define MAX_STEPS 50
extern std::chrono::high_resolution_clock::time_point g_timestamps[MAX_STEPS];
extern std::vector<std::string> g_stepNames;
extern int g_currentStep;

// 记录时间戳
void log_record_time(const char* stepName);

// 打印所有步骤的耗时
void log_print_timing();

// 初始化性能分析
void log_perf_init();

// 启用/禁用性能日志
void log_enable_perf_log(bool enable);
bool log_is_perf_log_enabled();

#endif  // OCR_LOG_H
