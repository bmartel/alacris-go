//go:build desktop && darwin

#import <Cocoa/Cocoa.h>

extern void goMenuInvoke(int id);

static NSStatusItem *appTrayItem;
static NSMenu *appTrayMenu;

@interface GoTrayTarget : NSObject
@end
@implementation GoTrayTarget
- (void)invoke:(id)sender {
    goMenuInvoke((int)[(NSMenuItem *)sender tag]);
}
@end

static GoTrayTarget *appTrayTarget;

void appTrayInstall(const char *title, const char *tooltip) {
    @autoreleasepool {
        if (!appTrayTarget) {
            appTrayTarget = [GoTrayTarget new];
        }
        appTrayItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        NSString *t = [NSString stringWithUTF8String:title ? title : "•"];
        appTrayItem.button.title = t;
        if (tooltip && tooltip[0]) {
            appTrayItem.button.toolTip = [NSString stringWithUTF8String:tooltip];
        }
        appTrayMenu = [[NSMenu alloc] init];
    }
}

void appTrayAddItem(int ident, const char *title) {
    @autoreleasepool {
        NSString *t = [NSString stringWithUTF8String:title ? title : ""];
        NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:t action:@selector(invoke:) keyEquivalent:@""];
        [item setTarget:appTrayTarget];
        [item setTag:ident];
        [appTrayMenu addItem:item];
    }
}

void appTrayAddSeparator(void) {
    [appTrayMenu addItem:[NSMenuItem separatorItem]];
}

void appTrayFinish(void) {
    [appTrayItem setMenu:appTrayMenu];
}
