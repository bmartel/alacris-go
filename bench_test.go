package alacris

import (
	"context"
	"io"
	"testing"

	"github.com/a-h/templ"
)

type benchItem struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

func BenchmarkElementScalarProps(b *testing.B) {
	ctx := templ.InitializeContext(context.Background())
	b.ReportAllocs()
	for b.Loop() {
		el := E("user-card").
			Prop("name", "Ada").
			Prop("age", 36).
			Prop("ratio", 1.5).
			Prop("active", true)
		if err := el.Render(ctx, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkElementJSONProp(b *testing.B) {
	items := make([]benchItem, 100)
	for i := range items {
		items[i] = benchItem{ID: i, Text: "something to do"}
	}
	ctx := templ.InitializeContext(context.Background())

	b.ReportAllocs()
	for b.Loop() {
		if err := E("ala-todo-list").Prop("items", items).Render(ctx, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

// Element.set scans linearly to keep first-set order with last-set value, so
// this is where an element with a lot of props would show it.
func BenchmarkElementManyProps(b *testing.B) {
	names := make([]string, 40)
	for i := range names {
		names[i] = "prop" + itoa(i)
	}
	ctx := templ.InitializeContext(context.Background())

	b.ReportAllocs()
	for b.Loop() {
		el := E("x-y")
		for i, n := range names {
			el.Prop(n, i)
		}
		if err := el.Render(ctx, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeProp(b *testing.B) {
	cases := map[string]any{
		"string": "hello",
		"int":    42,
		"bool":   true,
		"float":  1.5,
		"slice":  []string{"a", "b", "c"},
		"struct": benchItem{ID: 1, Text: "x"},
	}
	for name, v := range cases {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, _, err := EncodeProp(v); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
