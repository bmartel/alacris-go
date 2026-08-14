package live

import (
	"log/slog"
	"net/http"
	"testing"
)

// discardResponse is the cheapest possible ResponseWriter, so the benchmark
// measures the frame encoding rather than a recorder's bookkeeping.
type discardResponse struct{ h http.Header }

func (d *discardResponse) Header() http.Header         { return d.h }
func (d *discardResponse) Write(p []byte) (int, error) { return len(p), nil }
func (d *discardResponse) WriteHeader(int)             {}

// BenchmarkWriteFrame is the downstream hot path: one frame, encoded and
// written, exactly as serveStream does it per channel receive.
func BenchmarkWriteFrame(b *testing.B) {
	w := &discardResponse{h: http.Header{}}
	log := slog.New(slog.DiscardHandler)
	patches := []Patch{
		Prop("cart", "count", 3),
		Prop("cart", "total", 42.5),
		Class("cart", "busy", false),
	}
	b.ReportAllocs()
	for b.Loop() {
		if !writeFrame(w, patches, log) {
			b.Fatal("writeFrame reported a failed write")
		}
	}
}
