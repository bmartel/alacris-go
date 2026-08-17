//go:build desktop && !windows && !darwin

package app

func registerShortcut(*Window, string, func()) error { return ErrNoShortcut }

func unregisterShortcut(*Window, string) error { return nil }
