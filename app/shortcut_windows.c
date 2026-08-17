//go:build desktop && windows

#define WIN32_LEAN_AND_MEAN
#include <windows.h>

int appWinHotkeyRegister(void *hwnd, int id, int vk, int mods) {
    if (!hwnd) return 0;
    return RegisterHotKey((HWND)hwnd, id, (UINT)mods, (UINT)vk) ? 1 : 0;
}

void appWinHotkeyUnregister(void *hwnd, int id) {
    if (hwnd) UnregisterHotKey((HWND)hwnd, id);
}
