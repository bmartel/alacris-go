//go:build desktop && darwin

#import <Cocoa/Cocoa.h>

void appWindowMinimize(void *win) {
    if (!win) return;
    [(__bridge NSWindow *)win miniaturize:nil];
}

void appWindowMaximize(void *win) {
    if (!win) return;
    NSWindow *w = (__bridge NSWindow *)win;
    if (![w isZoomed]) {
        [w zoom:nil];
    }
}

void appWindowUnmaximize(void *win) {
    if (!win) return;
    NSWindow *w = (__bridge NSWindow *)win;
    if ([w isZoomed]) {
        [w zoom:nil];
    }
}

void appWindowFullscreen(void *win) {
    if (!win) return;
    NSWindow *w = (__bridge NSWindow *)win;
    if (([w styleMask] & NSWindowStyleMaskFullScreen) == 0) {
        [w toggleFullScreen:nil];
    }
}

void appWindowUnfullscreen(void *win) {
    if (!win) return;
    NSWindow *w = (__bridge NSWindow *)win;
    if (([w styleMask] & NSWindowStyleMaskFullScreen) != 0) {
        [w toggleFullScreen:nil];
    }
}

void appWindowShow(void *win) {
    if (!win) return;
    [(__bridge NSWindow *)win makeKeyAndOrderFront:nil];
}

void appWindowHide(void *win) {
    if (!win) return;
    [(__bridge NSWindow *)win orderOut:nil];
}

void appWindowFocus(void *win) {
    if (!win) return;
    [NSApp activateIgnoringOtherApps:YES];
    [(__bridge NSWindow *)win makeKeyAndOrderFront:nil];
}

void appWindowSetAlwaysOnTop(void *win, int on) {
    if (!win) return;
    NSWindow *w = (__bridge NSWindow *)win;
    [w setLevel:on ? NSFloatingWindowLevel : NSNormalWindowLevel];
}

void appWindowSetDecorations(void *win, int on) {
    if (!win) return;
    NSWindow *w = (__bridge NSWindow *)win;
    NSWindowStyleMask mask = [w styleMask];
    if (on) {
        mask |= NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
                NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable;
    } else {
        mask &= ~(NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
                  NSWindowStyleMaskMiniaturizable);
    }
    [w setStyleMask:mask];
}

void appWindowSetPosition(void *win, int x, int y) {
    if (!win) return;
    NSWindow *w = (__bridge NSWindow *)win;
    NSRect f = [w frame];
    f.origin.x = x;
    f.origin.y = y;
    [w setFrame:f display:YES];
}

void appWindowPosition(void *win, int *x, int *y) {
    if (!win || !x || !y) return;
    NSRect f = [(__bridge NSWindow *)win frame];
    *x = (int)f.origin.x;
    *y = (int)f.origin.y;
}

void appWindowCenter(void *win) {
    if (!win) return;
    [(__bridge NSWindow *)win center];
}

void appWindowSetBadge(const char *s) {
    @autoreleasepool {
        NSString *t = s ? [NSString stringWithUTF8String:s] : @"";
        [[NSApp dockTile] setBadgeLabel:t];
    }
}
