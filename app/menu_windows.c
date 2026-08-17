//go:build desktop && windows

#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

extern void goMenuInvoke(int id);
extern void goHotkey(int id);
extern int appTrayHandle(UINT msg, WPARAM wParam, LPARAM lParam);

static HMENU menuStack[16];
static int menuDepth;
WNDPROC appOldWndProc;
HWND appMenuHwnd;

static LRESULT CALLBACK appMenuWndProc(HWND h, UINT msg, WPARAM wParam, LPARAM lParam);

void appWinSubclass(void *hwnd) {
    appMenuHwnd = (HWND)hwnd;
    if (appMenuHwnd && !appOldWndProc) {
        appOldWndProc = (WNDPROC)SetWindowLongPtr(appMenuHwnd, GWLP_WNDPROC, (LONG_PTR)appMenuWndProc);
    }
}

static LRESULT CALLBACK appMenuWndProc(HWND h, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (appTrayHandle(msg, wParam, lParam)) {
        return 0;
    }
    if (msg == WM_COMMAND) {
        goMenuInvoke((int)LOWORD(wParam));
        return 0;
    }
    if (msg == WM_HOTKEY) {
        goHotkey((int)wParam);
        return 0;
    }
    if (appOldWndProc) {
        return CallWindowProc(appOldWndProc, h, msg, wParam, lParam);
    }
    return DefWindowProc(h, msg, wParam, lParam);
}

void appWinInstallMenu(void *hwnd) {
    appWinSubclass(hwnd);
    menuDepth = 0;
    menuStack[0] = CreateMenu();
}

static HMENU appWinCurrent(void) {
    return menuStack[menuDepth];
}

void appWinBeginSubmenu(const char *title) {
    HMENU sub = CreatePopupMenu();
    AppendMenuA(appWinCurrent(), MF_POPUP, (UINT_PTR)sub, title ? title : "");
    if (menuDepth < 15) {
        menuDepth++;
        menuStack[menuDepth] = sub;
    }
}

void appWinEndSubmenu(void) {
    if (menuDepth > 0) {
        menuDepth--;
    }
}

void appWinAddSeparator(void) {
    AppendMenuA(appWinCurrent(), MF_SEPARATOR, 0, NULL);
}

void appWinAddMenuItem(int id, const char *title, const char *key, int shift, int alt, int ctrl) {
    char buf[256];
    const char *t = title ? title : "";
    if (key && key[0]) {
        snprintf(buf, sizeof(buf), "%s\t%s%s%s%s", t,
                 ctrl ? "Ctrl+" : "",
                 shift ? "Shift+" : "",
                 alt ? "Alt+" : "",
                 key);
        t = buf;
    }
    AppendMenuA(appWinCurrent(), MF_STRING, (UINT_PTR)id, t);
}

void appWinAddRoleQuit(int id, const char *title) {
    appWinAddMenuItem(id, title, "q", 0, 0, 1);
}

void appWinFinishMenu(void *hwnd) {
    if (hwnd) {
        SetMenu((HWND)hwnd, menuStack[0]);
        DrawMenuBar((HWND)hwnd);
    }
}

void appWinRoleCut(void) { SendMessage(GetFocus(), WM_CUT, 0, 0); }
void appWinRoleCopy(void) { SendMessage(GetFocus(), WM_COPY, 0, 0); }
void appWinRolePaste(void) { SendMessage(GetFocus(), WM_PASTE, 0, 0); }
void appWinRoleSelectAll(void) {
    SendMessage(GetFocus(), EM_SETSEL, 0, -1);
}
