package appmeta

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// lookPath is exec.LookPath, replaced in tests.
var lookPath = exec.LookPath

var runCmd = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Pack runs optional OS tools on a tree BundleDir already wrote: codesign,
// hdiutil, nfpm, go-winres. Missing tools are skipped, not errors.
func Pack(c Config, dest, goos string) (string, error) {
	if goos == "" {
		goos = runtime.GOOS
	}
	switch goos {
	case "darwin":
		return packDarwin(c, dest)
	case "linux":
		return packLinux(c, dest)
	case "windows":
		return packWindows(c, dest)
	default:
		return dest, nil
	}
}

func packDarwin(c Config, dest string) (string, error) {
	appPath := dest
	if !strings.HasSuffix(appPath, ".app") {
		appPath = filepath.Join(dest, c.Name+".app")
	}
	id := c.Bundle.MacOS.SigningIdentity
	if id != "" {
		if _, err := lookPath("codesign"); err == nil {
			args := []string{"--force", "--sign", id, "--deep", "--options", "runtime", appPath}
			if err := runCmd("codesign", args...); err != nil {
				return "", fmt.Errorf("appmeta: codesign: %w", err)
			}
		}
	}
	if c.Bundle.MacOS.Notarize {
		if profile := os.Getenv("ALACRIS_NOTARY_PROFILE"); profile != "" {
			if _, err := lookPath("xcrun"); err == nil {
				zip := appPath + ".zip"
				if err := runCmd("ditto", "-c", "-k", "--keepParent", appPath, zip); err != nil {
					return "", fmt.Errorf("appmeta: ditto: %w", err)
				}
				if err := runCmd("xcrun", "notarytool", "submit", zip, "--keychain-profile", profile, "--wait"); err != nil {
					return "", fmt.Errorf("appmeta: notarytool: %w", err)
				}
				_ = runCmd("xcrun", "stapler", "staple", appPath)
			}
		}
	}
	if _, err := lookPath("hdiutil"); err == nil {
		dmg := strings.TrimSuffix(appPath, ".app") + ".dmg"
		if err := runCmd("hdiutil", "create", "-volname", c.Name, "-srcfolder", appPath, "-ov", "-format", "UDZO", dmg); err != nil {
			return appPath, fmt.Errorf("appmeta: hdiutil: %w", err)
		}
		return dmg, nil
	}
	return appPath, nil
}

func packLinux(c Config, dest string) (string, error) {
	if _, err := lookPath("nfpm"); err != nil {
		return dest, nil
	}
	cfg := filepath.Join(dest, "nfpm.yaml")
	out := dest
	if err := runCmd("nfpm", "package", "-f", cfg, "-p", "deb", "-t", out); err != nil {
		return dest, fmt.Errorf("appmeta: nfpm: %w", err)
	}
	return dest, nil
}

func packWindows(c Config, dest string) (string, error) {
	if _, err := lookPath("go-winres"); err != nil {
		return dest, nil
	}
	exe := filepath.Join(dest, c.BinaryName()+".exe")
	ico := filepath.Join(dest, c.BinaryName()+".ico")
	args := []string{"patch", "--in", exe}
	if _, err := os.Stat(ico); err == nil {
		args = append(args, "--ico", ico)
	}
	if err := runCmd("go-winres", args...); err != nil {
		// go-winres subcommands vary; a missing flag is not fatal.
		return dest, nil
	}
	return dest, nil
}
