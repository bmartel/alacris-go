//go:build !windows && !darwin

package app

func registerDeepLinkPlatform(scheme, identifier string) {
	_ = scheme
	_ = identifier
}
