package live

import (
	"io"
	"log/slog"
	"testing"
)

func quietServer(b *testing.B, opts ...Options) *Server {
	b.Helper()
	o := Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if len(opts) > 0 {
		o = opts[0]
		o.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	srv := New(o)
	b.Cleanup(srv.Close)
	return srv
}

// Broadcast fans one set of patches out to every session, and each stream
// encodes them again on its way out. This is what says whether that repeated
// encoding is worth removing.
func BenchmarkBroadcast(b *testing.B) {
	for _, sessions := range []int{1, 100, 1000} {
		b.Run(itoa(sessions)+"-sessions", func(b *testing.B) {
			srv := quietServer(b, Options{MaxSessions: sessions * 2})

			// Attached readers, so patches go out rather than buffering.
			for i := 0; i < sessions; i++ {
				sess := newSession(srv)
				patches, _, release, err := sess.subscribe()
				if err != nil {
					b.Fatal(err)
				}
				b.Cleanup(release)
				go func() {
					for range patches {
					}
				}()
			}

			patch := Prop("todos", "items", []map[string]any{{"id": 1, "text": "x"}})
			b.ReportAllocs()
			for b.Loop() {
				srv.Broadcast(patch)
			}
		})
	}
}

// Sessions() copies the map under a read lock, which every broadcast and every
// sweep of the collector pays for.
func BenchmarkSessions(b *testing.B) {
	srv := quietServer(b, Options{MaxSessions: 20000})
	for i := 0; i < 10000; i++ {
		newSession(srv)
	}

	b.ReportAllocs()
	for b.Loop() {
		if len(srv.Sessions()) == 0 {
			b.Fatal("no sessions")
		}
	}
}

// NewSession at the cap is the overload path: every render past MaxSessions
// evicts. This is what says whether eviction amplifies the load the cap
// exists to absorb.
func BenchmarkNewSessionAtCap(b *testing.B) {
	for _, cap := range []int{1000, 10000} {
		b.Run(itoa(cap), func(b *testing.B) {
			srv := quietServer(b, Options{MaxSessions: cap})
			for i := 0; i < cap; i++ {
				newSession(srv)
			}

			b.ReportAllocs()
			for b.Loop() {
				newSession(srv)
			}
		})
	}
}

func BenchmarkSendToAttachedSession(b *testing.B) {
	srv := quietServer(b)
	sess := newSession(srv)
	patches, _, release, err := sess.subscribe()
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(release)
	go func() {
		for range patches {
		}
	}()

	b.ReportAllocs()
	for b.Loop() {
		sess.Element("todos").Set("count", 1)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
