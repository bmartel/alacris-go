package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// DataDir is the per-app directory for durable files (config, lock, sockets).
// identifier is a reverse-DNS id, the same value as Options.Identifier.
func DataDir(identifier string) (string, error) {
	if identifier == "" {
		return "", fmt.Errorf("app: DataDir requires an identifier")
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, identifier)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// CacheDir is the per-app directory for disposable files.
func CacheDir(identifier string) (string, error) {
	if identifier == "" {
		return "", fmt.Errorf("app: CacheDir requires an identifier")
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, identifier)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
