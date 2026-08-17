//go:build desktop && darwin

#import <Cocoa/Cocoa.h>

extern void goMenuInvoke(int id);

static NSMutableArray *menuStack;
static NSMenu *menubar;
static id menuTarget;

@interface GoMenuTarget : NSObject
@end
@implementation GoMenuTarget
- (void)invoke:(id)sender {
    goMenuInvoke((int)[(NSMenuItem *)sender tag]);
}
@end

void appInstallMenu(void) {
    @autoreleasepool {
        menubar = [[NSMenu alloc] init];
        menuStack = [NSMutableArray arrayWithObject:menubar];
        if (!menuTarget) {
            menuTarget = [GoMenuTarget new];
        }
    }
}

static NSMenu *appCurrentMenu(void) {
    return [menuStack lastObject];
}

void appBeginSubmenu(const char *title) {
    @autoreleasepool {
        NSString *t = [NSString stringWithUTF8String:title ? title : ""];
        NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:t action:NULL keyEquivalent:@""];
        NSMenu *sub = [[NSMenu alloc] initWithTitle:t];
        [item setSubmenu:sub];
        [appCurrentMenu() addItem:item];
        [menuStack addObject:sub];
    }
}

void appEndSubmenu(void) {
    if ([menuStack count] > 1) {
        [menuStack removeLastObject];
    }
}

void appAddSeparator(void) {
    [appCurrentMenu() addItem:[NSMenuItem separatorItem]];
}

void appAddRoleItem(const char *title, const char *key, const char *selector) {
    @autoreleasepool {
        NSString *t = [NSString stringWithUTF8String:title ? title : ""];
        NSString *k = [NSString stringWithUTF8String:key ? key : ""];
        SEL sel = NSSelectorFromString([NSString stringWithUTF8String:selector ? selector : ""]);
        NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:t action:sel keyEquivalent:k];
        if (sel == @selector(terminate:)) {
            [item setTarget:NSApp];
        }
        [appCurrentMenu() addItem:item];
    }
}

void appAddMenuItem(int ident, const char *title, const char *key, int shift, int alt, int cmd) {
    @autoreleasepool {
        NSString *t = [NSString stringWithUTF8String:title ? title : ""];
        NSString *k = [NSString stringWithUTF8String:key ? key : ""];
        NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:t action:@selector(invoke:) keyEquivalent:k];
        [item setTarget:menuTarget];
        [item setTag:ident];
        NSEventModifierFlags flags = 0;
        if (cmd) {
            flags |= NSEventModifierFlagCommand;
        }
        if (shift) {
            flags |= NSEventModifierFlagShift;
        }
        if (alt) {
            flags |= NSEventModifierFlagOption;
        }
        if (flags) {
            [item setKeyEquivalentModifierMask:flags];
        }
        [appCurrentMenu() addItem:item];
    }
}

void appFinishMenu(void) {
    [NSApp setMainMenu:menubar];
}
