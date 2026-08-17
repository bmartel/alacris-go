//go:build desktop && windows

#define WIN32_LEAN_AND_MEAN
#include <windows.h>

void appWindowMinimize(void *win) {
    if (win) ShowWindow((HWND)win, SW_MINIMIZE);
}

void appWindowMaximize(void *win) {
    if (win) ShowWindow((HWND)win, SW_MAXIMIZE);
}

void appWindowUnmaximize(void *win) {
    if (win) ShowWindow((HWND)win, SW_RESTORE);
}

static WINDOWPLACEMENT appSavedPlacement;
static int appHavePlacement;

void appWindowFullscreen(void *win) {
    if (!win) return;
    HWND h = (HWND)win;
    appHavePlacement = GetWindowPlacement(h, &appSavedPlacement);
    HMONITOR mon = MonitorFromWindow(h, MONITOR_DEFAULTTONEAREST);
    MONITORINFO mi;
    mi.cbSize = sizeof(mi);
    if (!GetMonitorInfo(mon, &mi)) return;
    SetWindowLongPtr(h, GWL_STYLE, WS_POPUP | WS_VISIBLE);
    SetWindowPos(h, HWND_TOP, mi.rcMonitor.left, mi.rcMonitor.top,
                 mi.rcMonitor.right - mi.rcMonitor.left,
                 mi.rcMonitor.bottom - mi.rcMonitor.top,
                 SWP_FRAMECHANGED);
}

void appWindowUnfullscreen(void *win) {
    if (!win) return;
    HWND h = (HWND)win;
    SetWindowLongPtr(h, GWL_STYLE, WS_OVERLAPPEDWINDOW | WS_VISIBLE);
    if (appHavePlacement) {
        SetWindowPlacement(h, &appSavedPlacement);
    }
    SetWindowPos(h, NULL, 0, 0, 0, 0,
                 SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_FRAMECHANGED);
}

void appWindowShow(void *win) {
    if (win) ShowWindow((HWND)win, SW_SHOW);
}

void appWindowHide(void *win) {
    if (win) ShowWindow((HWND)win, SW_HIDE);
}

void appWindowFocus(void *win) {
    if (!win) return;
    HWND h = (HWND)win;
    ShowWindow(h, SW_SHOW);
    SetForegroundWindow(h);
    BringWindowToTop(h);
}

void appWindowSetAlwaysOnTop(void *win, int on) {
    if (!win) return;
    SetWindowPos((HWND)win, on ? HWND_TOPMOST : HWND_NOTOPMOST, 0, 0, 0, 0,
                 SWP_NOMOVE | SWP_NOSIZE);
}

void appWindowSetDecorations(void *win, int on) {
    if (!win) return;
    HWND h = (HWND)win;
    LONG style = GetWindowLong(h, GWL_STYLE);
    if (on) {
        style |= WS_CAPTION | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX | WS_SYSMENU;
    } else {
        style &= ~(WS_CAPTION | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX | WS_SYSMENU);
    }
    SetWindowLong(h, GWL_STYLE, style);
    SetWindowPos(h, NULL, 0, 0, 0, 0,
                 SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_FRAMECHANGED);
}

void appWindowSetPosition(void *win, int x, int y) {
    if (win) SetWindowPos((HWND)win, NULL, x, y, 0, 0, SWP_NOSIZE | SWP_NOZORDER);
}

void appWindowPosition(void *win, int *x, int *y) {
    if (!win || !x || !y) return;
    RECT r;
    if (!GetWindowRect((HWND)win, &r)) return;
    *x = r.left;
    *y = r.top;
}

void appWindowCenter(void *win) {
    if (!win) return;
    HWND h = (HWND)win;
    RECT r;
    GetWindowRect(h, &r);
    int w = r.right - r.left;
    int ht = r.bottom - r.top;
    int sw = GetSystemMetrics(SM_CXSCREEN);
    int sh = GetSystemMetrics(SM_CYSCREEN);
    SetWindowPos(h, NULL, (sw - w) / 2, (sh - ht) / 2, 0, 0, SWP_NOSIZE | SWP_NOZORDER);
}
