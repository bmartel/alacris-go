package alacris

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"
)

func TestAttrName(t *testing.T) {
	// These cases are transcribed from define.js's kebab, quirks included: the
	// Go side has to produce the name the element is actually observing.
	cases := map[string]string{
		"name":     "name",
		"maxCount": "max-count",
		"isOpen":   "is-open",
		"a":        "a",
		"aB":       "a-b",
		"Name":     "-name",  // a leading capital really does produce a leading dash
		"URL":      "-u-r-l", // and an acronym really does expand letter by letter
		"dataURL":  "data-u-r-l",
		"":         "",
	}
	for in, want := range cases {
		if got := AttrName(in); got != want {
			t.Errorf("AttrName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEncodeProp(t *testing.T) {
	type point struct {
		X int `json:"x"`
		Y int `json:"y"`
	}

	cases := []struct {
		name string
		in   any
		want string
		omit bool
	}{
		{"string", "Ada", "Ada", false},
		{"empty string", "", "", false},

		// Booleans are written out either way. Presence/absence is not usable:
		// removing an attribute runs coerce(null, default), which returns false
		// even when the declared default is true.
		{"true", true, "true", false},
		{"false", false, "false", false},

		{"int", 36, "36", false},
		{"negative int", -1, "-1", false},
		{"uint", uint(7), "7", false},
		{"float", 1.5, "1.5", false},
		{"float integral", 3.0, "3", false},
		{"float exponent", 1e21, "1e+21", false},

		{"slice", []string{"a", "b"}, `["a","b"]`, false},
		{"empty slice", []string{}, `[]`, false},
		{"struct", point{1, 2}, `{"x":1,"y":2}`, false},
		{"pointer to struct", &point{1, 2}, `{"x":1,"y":2}`, false},
		{"map", map[string]int{"b": 2, "a": 1}, `{"a":1,"b":2}`, false},
		{"raw json", json.RawMessage(`{"already":"encoded"}`), `{"already":"encoded"}`, false},

		{"time", time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC), "2026-08-08T12:00:00Z", false},
		{"duration in ms", 1500 * time.Millisecond, "1500", false},

		{"nil", nil, "", true},
		{"nil pointer", (*point)(nil), "", true},
		{"nil slice", []string(nil), "", true},
		{"nil map", map[string]int(nil), "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok, err := EncodeProp(c.in)
			if err != nil {
				t.Fatalf("EncodeProp(%#v): %v", c.in, err)
			}
			if ok == c.omit {
				t.Fatalf("EncodeProp(%#v) ok = %v, want %v", c.in, ok, !c.omit)
			}
			if ok && got != c.want {
				t.Errorf("EncodeProp(%#v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestEncodePropRejectsUnrepresentableNumbers(t *testing.T) {
	// A prop is coerced with `+v` on the other side, so anything past 2^53-1
	// arrives as a different number. Rounding an identifier in silence is the
	// worst possible outcome, so it is an error.
	for _, v := range []any{int64(MaxSafeInteger + 1), int64(-MaxSafeInteger - 1), uint64(1) << 62} {
		if _, _, err := EncodeProp(v); !errors.Is(err, ErrUnsafeInteger) {
			t.Errorf("EncodeProp(%v) error = %v, want ErrUnsafeInteger", v, err)
		}
	}
	if _, _, err := EncodeProp(int64(MaxSafeInteger)); err != nil {
		t.Errorf("EncodeProp(MaxSafeInteger): %v", err)
	}

	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, _, err := EncodeProp(v); !errors.Is(err, ErrNonFiniteFloat) {
			t.Errorf("EncodeProp(%v) error = %v, want ErrNonFiniteFloat", v, err)
		}
	}
}

func TestEncodeAttrFollowsHTMLBooleanRules(t *testing.T) {
	// Ordinary HTML attributes are the opposite of props: presence is the
	// value, so false has to disappear rather than render as "false".
	s, bare, ok, err := EncodeAttr(true)
	if err != nil || !ok || !bare || s != "" {
		t.Errorf("EncodeAttr(true) = (%q, %v, %v, %v)", s, bare, ok, err)
	}
	if _, _, ok, _ := EncodeAttr(false); ok {
		t.Error("EncodeAttr(false) should omit the attribute")
	}
	if _, _, ok, _ := EncodeAttr(nil); ok {
		t.Error("EncodeAttr(nil) should omit the attribute")
	}
}

// A named float32 went through the reflect path, which formatted it as the
// float64 it had been widened to: 0.1 came out as 0.10000000149011612. The
// direct float32 case had always been right, so this only ever bit types that
// wrapped one.
func TestEncodePropNamedFloatTypes(t *testing.T) {
	type celsius float32
	type ratio float64

	cases := []struct {
		in   any
		want string
	}{
		{float32(0.1), "0.1"},
		{celsius(0.1), "0.1"},
		{celsius(36.6), "36.6"},
		{ratio(0.1), "0.1"},
		{float64(0.1), "0.1"},
	}

	for _, c := range cases {
		got, _, err := EncodeProp(c.in)
		if err != nil {
			t.Fatalf("EncodeProp(%#v): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("EncodeProp(%#v) = %q, want %q", c.in, got, c.want)
		}
	}
}
