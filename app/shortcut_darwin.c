//go:build desktop && darwin

#include <Carbon/Carbon.h>
#include <stdint.h>

extern void goHotkey(int id);

static OSStatus appHotkeyHandler(EventHandlerCallRef next, EventRef event, void *data) {
    EventHotKeyID hk;
    GetEventParameter(event, kEventParamDirectObject, typeEventHotKeyID, NULL, sizeof(hk), NULL, &hk);
    goHotkey((int)hk.id);
    return noErr;
}

static int appHotkeyReady;

static void appHotkeyEnsure(void) {
    if (appHotkeyReady) return;
    EventTypeSpec spec = {kEventClassKeyboard, kEventHotKeyPressed};
    InstallApplicationEventHandler(NewEventHandlerUPP(appHotkeyHandler), 1, &spec, NULL, NULL);
    appHotkeyReady = 1;
}

int appHotkeyRegister(int id, int key, int shift, int alt, int cmd) {
    appHotkeyEnsure();
    EventHotKeyRef ref;
    EventHotKeyID hk;
    hk.signature = 'alcr';
    hk.id = (UInt32)id;
    UInt32 mods = 0;
    if (shift) mods |= shiftKey;
    if (alt) mods |= optionKey;
    if (cmd) mods |= cmdKey;
    UInt32 vk = (UInt32)key;
    if (key >= 'A' && key <= 'Z') key += 32;
    if (key >= 'a' && key <= 'z') {
        static const UInt32 t[26] = {
            0x00, 0x0B, 0x08, 0x02, 0x0E, 0x03, 0x05, 0x04, 0x22, 0x26,
            0x28, 0x25, 0x2E, 0x2D, 0x1F, 0x23, 0x0C, 0x0F, 0x01, 0x11,
            0x20, 0x09, 0x0D, 0x07, 0x10, 0x06
        };
        vk = t[key - 'a'];
    }
    OSStatus st = RegisterEventHotKey(vk, mods, hk, GetApplicationEventTarget(), 0, &ref);
    return st == noErr ? 1 : 0;
}

void appHotkeyUnregister(int id) {
    (void)id;
}
