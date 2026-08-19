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

// The style values match the Titlebar constants in app.go: 0 native,
// 1 inset, 2 hidden.
void appWindowSetTitlebar(void *win, int style) {
    if (!win) return;
    NSWindow *w = (__bridge NSWindow *)win;
    NSWindowStyleMask mask = [w styleMask];

    // Titled stays set for every style, including hidden. A borderless
    // NSWindow answers NO to canBecomeKeyWindow unless it is subclassed, and
    // this one is the webview's — so dropping Titled to hide the bar takes the
    // keyboard with it, and the window comes up looking right and refusing to
    // let anyone type. Hiding is done by making the bar transparent and taking
    // the buttons out instead.
    mask |= NSWindowStyleMaskTitled;

    BOOL hidden = (style == 2);
    BOOL bare = (style != 0);

    if (bare) {
        mask |= NSWindowStyleMaskFullSizeContentView;
    } else {
        mask &= ~NSWindowStyleMaskFullSizeContentView;
    }
    [w setStyleMask:mask];

    [w setTitlebarAppearsTransparent:bare];
    [w setTitleVisibility:bare ? NSWindowTitleHidden
                               : NSWindowTitleVisible];

    // The traffic lights stay for inset — they are the reason to prefer it —
    // and go for hidden, where the page draws its own.
    NSWindowButton buttons[3] = {NSWindowCloseButton,
                                 NSWindowMiniaturizeButton,
                                 NSWindowZoomButton};
    for (int i = 0; i < 3; i++) {
        NSButton *b = [w standardWindowButton:buttons[i]];
        if (b) [b setHidden:hidden];
    }
}

void appWindowBeginDrag(void *win) {
    if (!win) return;
    NSWindow *w = (__bridge NSWindow *)win;
    NSEvent *e = [NSApp currentEvent];
    // Only a real mouse-down can start a drag; anything else would either be
    // ignored or leave the window stuck to the pointer.
    if (e && [e type] == NSEventTypeLeftMouseDown) {
        [w performWindowDragWithEvent:e];
    }
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
