//go:build desktop && darwin

#import <Cocoa/Cocoa.h>

extern void goDeepLink(char *url);

@interface GoDeepLinkTarget : NSObject
@end
@implementation GoDeepLinkTarget
- (void)handleURLEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)reply {
    (void)reply;
    NSString *url = [[event paramDescriptorForKeyword:keyDirectObject] stringValue];
    if (url) {
        goDeepLink((char *)[url UTF8String]);
    }
}
@end

static GoDeepLinkTarget *appDeepLinkTarget;

void appDeepLinkInstall(void) {
    @autoreleasepool {
        if (!appDeepLinkTarget) {
            appDeepLinkTarget = [GoDeepLinkTarget new];
        }
        [[NSAppleEventManager sharedAppleEventManager]
            setEventHandler:appDeepLinkTarget
            andSelector:@selector(handleURLEvent:withReplyEvent:)
            forEventClass:kInternetEventClass
            andEventID:kAEGetURL];
    }
}
