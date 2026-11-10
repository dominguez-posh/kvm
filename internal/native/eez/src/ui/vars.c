#include <string.h>
#include <stdio.h>
#include <lvgl.h>
#include "vars.h"

char app_version[100] = { 0 };
char system_version[100] = { 0 };
char lvgl_version[32] = { 0 };
char main_screen[32] = "home_screen";

const char *get_var_app_version() {
    return app_version;
}

const char *get_var_system_version() {
    return system_version;
}

const char *get_var_lvgl_version() {
    if (lvgl_version[0] == '\0') {
        char buf[32];
        sprintf(buf, "%d.%d.%d", LVGL_VERSION_MAJOR, LVGL_VERSION_MINOR, LVGL_VERSION_PATCH);
        
        
        strncpy(lvgl_version, buf, sizeof(lvgl_version) / sizeof(char));
        app_version[sizeof(lvgl_version) / sizeof(char) - 1] = 0;
    }
    return lvgl_version;
}

void set_var_app_version(const char *value) {
    strncpy(app_version, value, sizeof(app_version) / sizeof(char));
    app_version[sizeof(app_version) / sizeof(char) - 1] = 0;
}

void set_var_system_version(const char *value) {
    strncpy(system_version, value, sizeof(system_version) / sizeof(char));
    system_version[sizeof(system_version) / sizeof(char) - 1] = 0;
}

void set_var_lvgl_version(const char *value) {}

void set_var_main_screen(const char *value) {
    strncpy(main_screen, value, sizeof(main_screen) / sizeof(char));
    main_screen[sizeof(main_screen) / sizeof(char) - 1] = 0;
}

const char *get_var_main_screen() {
    return main_screen;
}


char claude_5h_pct[8] = { 0 };

const char *get_var_claude_5h_pct() {
    return claude_5h_pct;
}

void set_var_claude_5h_pct(const char *value) {
    strncpy(claude_5h_pct, value, sizeof(claude_5h_pct) - 1);
    claude_5h_pct[sizeof(claude_5h_pct) - 1] = 0;
}

char claude_7d_pct[8] = { 0 };

const char *get_var_claude_7d_pct() {
    return claude_7d_pct;
}

void set_var_claude_7d_pct(const char *value) {
    strncpy(claude_7d_pct, value, sizeof(claude_7d_pct) - 1);
    claude_7d_pct[sizeof(claude_7d_pct) - 1] = 0;
}

char system_cpu_pct[8] = { 0 };

const char *get_var_system_cpu_pct() {
    return system_cpu_pct;
}

void set_var_system_cpu_pct(const char *value) {
    strncpy(system_cpu_pct, value, sizeof(system_cpu_pct) - 1);
    system_cpu_pct[sizeof(system_cpu_pct) - 1] = 0;
}

char system_ram_pct[8] = { 0 };

const char *get_var_system_ram_pct() {
    return system_ram_pct;
}

void set_var_system_ram_pct(const char *value) {
    strncpy(system_ram_pct, value, sizeof(system_ram_pct) - 1);
    system_ram_pct[sizeof(system_ram_pct) - 1] = 0;
}
