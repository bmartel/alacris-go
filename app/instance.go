package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrAlreadyRunning is returned by Run when SingleInstance is set and
// another process of this app holds the lock.
var ErrAlreadyRunning = errors.New("app: another instance is running")

type instanceLock struct {
	file *os.File
	ln   net.Listener
}

func osArgs() []string {
	if len(os.Args) < 2 {
		return nil
	}
	return append([]string(nil), os.Args[1:]...)
}

func looksLikeURL(s string) bool {
	return strings.Contains(s, "://")
}

func acquireInstance(id string, args []string, onSecond func([]string)) (*instanceLock, error) {
	dir, err := DataDir(id)
	if err != nil {
		return nil, err
	}
	lockPath := filepath.Join(dir, "instance.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(f); err != nil {
		port, _ := readInstancePort(f)
		_ = f.Close()
		if port != "" {
			_ = pingInstance(port, args)
		}
		return nil, ErrAlreadyRunning
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		unlockFile(f)
		_ = f.Close()
		return nil, err
	}
	_, p, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		_ = ln.Close()
		unlockFile(f)
		_ = f.Close()
		return nil, err
	}
	if err := writeInstancePort(f, p); err != nil {
		_ = ln.Close()
		unlockFile(f)
		_ = f.Close()
		return nil, err
	}
	lock := &instanceLock{file: f, ln: ln}
	go lock.serve(onSecond)
	return lock, nil
}

func (l *instanceLock) serve(onSecond func([]string)) {
	for {
		c, err := l.ln.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			var args []string
			if err := json.NewDecoder(io.LimitReader(c, 64<<10)).Decode(&args); err != nil {
				return
			}
			if onSecond != nil {
				onSecond(args)
			}
		}(c)
	}
}

func (l *instanceLock) release() {
	if l == nil {
		return
	}
	if l.ln != nil {
		_ = l.ln.Close()
	}
	if l.file != nil {
		unlockFile(l.file)
		_ = l.file.Close()
	}
}

func pingInstance(port string, args []string) error {
	c, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		return err
	}
	defer c.Close()
	return json.NewEncoder(c).Encode(args)
}

func writeInstancePort(f *os.File, port string) error {
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err := fmt.Fprintf(f, "%s\n", port)
	return err
}

func readInstancePort(f *os.File) (string, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return "", err
	}
	b, err := io.ReadAll(io.LimitReader(f, 32))
	if err != nil {
		return "", err
	}
	p := strings.TrimSpace(string(b))
	if _, err := strconv.Atoi(p); err != nil {
		return "", err
	}
	return p, nil
}
