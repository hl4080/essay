#ifndef __LOG_C_H__
#define __LOG_C_H__

#ifdef __cplusplus
extern "C" {
#endif

#define LOG_PATH "/var/log/log_c.log"
#define LOG_FORMAT_LEN 1024
#define LOG_TIME_LEN 100

enum LOG_LEVEL {
    LEVEL_DEBUG = 0,
    LEVEL_INFO,
    LEVEL_WARN,
    LEVEL_ERROR,
    LEVEL_MAX
};

void log(const char *func, int line, enum LOG_LEVEL type, const char *format, ...);
void log_init(void);
void log_uninit(void);

#define LOG_DEBUG(format, ...) log(__FUNCTION__, __LINE__, LEVEL_DEBUG, format, ##__VA_ARGS__);
#define LOG_INFO(format, ...) log(__FUNCTION__, __LINE__, LEVEL_INFO, format, ##__VA_ARGS__);
#define LOG_WARN(format, ...) log(__FUNCTION__, __LINE__, LEVEL_WARN, format, ##__VA_ARGS__);
#define LOG_ERR(format, ...) log(__FUNCTION__, __LINE__, LEVEL_ERROR, format, ##__VA_ARGS__);

#ifdef __cplusplus
}
#endif

#endif /*__LOG_C_H__*/