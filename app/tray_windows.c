//go:build desktop && windows

#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <shellapi.h>
#include <string.h>

extern void goMenuInvoke(int id);
extern void appWinSubclass(void *hwnd);

#define APP_TRAY_UID 1
#define APP_WM_TRAY (WM_APP + 1)

static NOTIFYICONDATAA appTrayData;
static HMENU appTrayMenu;
static HWND appTrayHwnd;

static void appTrayPopup(void) {
    if (!appTrayMenu || !appTrayHwnd) return;
    POINT p;
    GetCursorPos(&p);
    SetForegroundWindow(appTrayHwnd);
    TrackPopupMenu(appTrayMenu, TPM_BOTTOMALIGN | TPM_LEFTALIGN, p.x, p.y, 0, appTrayHwnd, NULL);
}

void appTrayInstall(void *hwnd, const char *tip) {
    appTrayHwnd = (HWND)hwnd;
    appTrayMenu = CreatePopupMenu();
    memset(&appTrayData, 0, sizeof(appTrayData));
    appTrayData.cbSize = sizeof(appTrayData);
    appTrayData.hWnd = appTrayHwnd;
    appTrayData.uID = APP_TRAY_UID;
    appTrayData.uFlags = NIF_MESSAGE | NIF_TIP | NIF_ICON;
    appTrayData.uCallbackMessage = APP_WM_TRAY;
    appTrayData.hIcon = LoadIcon(NULL, IDI_APPLICATION);
    if (tip) {
        strncpy(appTrayData.szTip, tip, sizeof(appTrayData.szTip) - 1);
    }
    Shell_NotifyIconA(NIM_ADD, &appTrayData);
    appWinSubclass(hwnd);
}

void appTrayAddItem(int id, const char *title) {
    if (appTrayMenu) {
        AppendMenuA(appTrayMenu, MF_STRING, (UINT_PTR)id, title ? title : "");
    }
}

void appTrayAddSeparator(void) {
    if (appTrayMenu) {
        AppendMenuA(appTrayMenu, MF_SEPARATOR, 0, NULL);
    }
}

void appTrayFinish(void) {}

/* Called from the subclassed WndProc via a exported helper. */
int appTrayHandle(UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == APP_WM_TRAY && (lParam == WM_RBUTTONUP || lParam == WM_LBUTTONUP)) {
        appTrayPopup();
        return 1;
    }
    return 0;
}
