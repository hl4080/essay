#include <pthread.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <time.h>
#include <unistd.h>
#include "log_c.h"

#ifdef __cplusplus
extern "C" {
#endif

#define BASE_YEAR 1900

FILE *g_file = NULL;
pthread_mutex_t g_file_mutex = PTHREAD_MUTEX_INITIALIZER;
pid_t g_current_pid = 0;
enum LOG_LEVEL g_level = LEVEL_INFO;

char *g_level_s[LEVEL_MAX] = {
    [LEVEL_DEBUG] = "DEBUG",
    [LEVEL_INFO] = "INFO",
    [LEVEL_WARN] = "WARN",
    [LEVEL_ERROR] = "ERROR"
};

static void log_level_set(void) {
    char *env;
    env = getenv("LOG_LEVEL");
    if(!env) {
        return;
    }
    if(!strcmp(env, "DEBUG")) {
        g_level = LEVEL_DEBUG;
    }else if(!strcmp(env, "INFO")) {
        g_level = LEVEL_INFO;
    }else if(!strcmp(env, "WARN")) {
        g_level = LEVEL_WARN;
    }else if(!strcmp(env, "ERROR")) {
        g_level = LEVEL_ERROR;
    }else g_level = LEVEL_INFO;
    return;
}

static bool log_level_check(enum LOG_LEVEL type) {
    return ((type >= g_level) && (type < LEVEL_MAX))? true: false;
}

void log_init(void) {
    int ret;
    pthread_mutex_lock(&g_file_mutex);
    if(!g_file) {
        g_file = fopen(LOG_PATH, "a");
        if(!g_file) {
            fprintf(stderr, "[LOG_C] %s(%d): [ERR]Failed to open log file\n", \
            __FUNCTION__, __LINE__);
        } else {
            ret = chmod(LOG_PATH, (S_IRUSR | S_IWUSR | S_IRGRP));
            if(ret) {
                fprintf(stderr, "[LOG_C] %s(%d): [ERR]Failed to chomod log file %d\n", \
                    __FUNCTION__, __LINE__, ret);
                fclose(g_file);
                g_file = NULL;
            }
        }
    }
    pthread_mutex_unlock(&g_file_mutex);
    g_current_pid = getpid();
    log_level_set();
}

void log_uninit(void) {
    pthread_mutex_lock(&g_file_mutex);
    if(g_file != NULL) {
        fclose(g_file);
        g_file = NULL;
    }
    pthread_mutex_unlock(&g_file_mutex);
}

void log(const char *func, int line, enum LOG_LEVEL type, const char *format, ...){
    va_list args;
    char file_format[LOG_FORMAT_LEN+1] = {0};
    FILE *f = g_file;
    time_t time_p;
    struct tm *localtime_p = NULL;
    char time_format[LOG_FORMAT_LEN+1] = {0};
    int ret;
    if(f == NULL) {
        f = stderr;
    }
    if(!log_level_check(type)) {
        return;
    }
    time(&time_p);
    localtime_p = localtime(&time_p);
    if(localtime_p != NULL) {
        ret = snprintf(time_format, sizeof(time_format), "%d-%02d-%02d %02d:%02d:%02d",
            (BASE_YEAR + localtime_p->tm_year), (1 + localtime_p->tm_mon), localtime_p->tm_mday, localtime_p->tm_hour,
            localtime_p->tm_min, localtime_p->tm_sec);
        if(ret < 0) {
            LOG_ERR("Failed to do snprintf, ret(%d)", ret);
            return;
        }
    }
    ret = snprintf(file_format, sizeof(file_format), "[%s] [PID %d] %s[%d] | %s | %s\n",
        time_format, g_current_pid, func, line, g_level_s[type], format);  
    va_start(args, format);
    vfprintf(f, file_format, args);
    va_end(args);
    fflush(f);
}

#ifdef __cplusplus
}
#endif