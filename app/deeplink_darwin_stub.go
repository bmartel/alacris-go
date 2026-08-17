//go:build darwin && !desktop

package app

func registerDeepLinkPlatform(scheme, identifier string) {
	_ = scheme
	_ = identifier
}
