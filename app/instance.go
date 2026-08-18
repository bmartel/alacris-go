package app

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
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
	file   *os.File
	ln     net.Listener
	secret string
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
		port, secret, _ := readInstanceInfo(f)
		_ = f.Close()
		if port != "" {
			_ = pingInstance(port, secret, args)
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
	// The listener is loopback TCP, which is not isolated per user — any local
	// user can connect to the port. The 0600 lock file, however, only the
	// owner can read, so a secret stored there authenticates a peer as the same
	// user. Without it a co-tenant could inject deep links or CLI args into
	// this app's second-instance handler.
	secret, err := newInstanceSecret()
	if err != nil {
		_ = ln.Close()
		unlockFile(f)
		_ = f.Close()
		return nil, err
	}
	if err := writeInstanceInfo(f, p, secret); err != nil {
		_ = ln.Close()
		unlockFile(f)
		_ = f.Close()
		return nil, err
	}
	lock := &instanceLock{file: f, ln: ln, secret: secret}
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
			r := bufio.NewReader(io.LimitReader(c, 64<<10))
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			// Constant-time so a co-tenant cannot recover the secret by timing
			// how far the comparison got.
			if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(line)), []byte(l.secret)) != 1 {
				return
			}
			var args []string
			if err := json.NewDecoder(r).Decode(&args); err != nil {
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

func pingInstance(port, secret string, args []string) error {
	c, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err := fmt.Fprintf(c, "%s\n", secret); err != nil {
		return err
	}
	return json.NewEncoder(c).Encode(args)
}

func newInstanceSecret() (string, error) {
	var b [18]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func writeInstanceInfo(f *os.File, port, secret string) error {
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err := fmt.Fprintf(f, "%s\n%s\n", port, secret)
	return err
}

func readInstanceInfo(f *os.File) (port, secret string, err error) {
	if _, err := f.Seek(0, 0); err != nil {
		return "", "", err
	}
	b, err := io.ReadAll(io.LimitReader(f, 128))
	if err != nil {
		return "", "", err
	}
	lines := strings.SplitN(strings.TrimSpace(string(b)), "\n", 2)
	port = strings.TrimSpace(lines[0])
	if _, err := strconv.Atoi(port); err != nil {
		return "", "", err
	}
	if len(lines) > 1 {
		secret = strings.TrimSpace(lines[1])
	}
	return port, secret, nil
}
