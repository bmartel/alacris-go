package alacris

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// MaxSafeInteger is JavaScript's Number.MAX_SAFE_INTEGER. Integers beyond this
// magnitude cannot survive the trip through alacris' numeric coercion (`+v`),
// which produces a float64. Encoding one is an error rather than a silent
// rounding: send large identifiers as strings instead.
const MaxSafeInteger = 1<<53 - 1

// ErrUnsafeInteger is returned when an integer prop cannot be represented
// exactly as a JavaScript number.
var ErrUnsafeInteger = errors.New("alacris: integer exceeds JavaScript's safe integer range; encode it as a string")

// ErrNonFiniteFloat is returned for NaN and ±Inf, which have no attribute form.
var ErrNonFiniteFloat = errors.New("alacris: NaN and Inf cannot be encoded as a prop")

// AttrName converts a component's JavaScript prop name to the attribute name
// alacris observes for it.
//
// It mirrors define.js exactly:
//
//	const kebab = s => s.replace(/[A-Z]/g, c => '-' + c.toLowerCase());
//
// including its sharp edges — a leading capital produces a leading dash, and
// an acronym expands letter by letter ("URL" becomes "-u-r-l"). Reproducing
// the quirks matters more than being tasteful: the attribute has to match what
// the element is listening for.
func AttrName(prop string) string {
	if !strings.ContainsFunc(prop, isASCIIUpper) {
		return prop
	}
	var b strings.Builder
	b.Grow(len(prop) + 4)
	for _, r := range prop {
		if isASCIIUpper(r) {
			b.WriteByte('-')
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isASCIIUpper(r rune) bool { return r >= 'A' && r <= 'Z' }

// EncodeProp renders v as the attribute text for an alacris prop.
//
// The encoding is chosen from v's Go type, and must line up with the type of
// the prop's default in the component's define() call, because that default is
// what drives coercion on the other side (define.js, coerce):
//
//	Go string, fmt.Stringer, time.Time  ->  text        (default: a string)
//	Go bool                             ->  "true"/"false" (default: a boolean)
//	Go integers and floats              ->  a number    (default: a number)
//	everything else                     ->  JSON        (default: an object or array)
//
// ok is false when v is nil or a nil pointer, meaning the attribute should be
// left off entirely so the component keeps its declared default.
func EncodeProp(v any) (s string, ok bool, err error) {
	switch t := v.(type) {
	case nil:
		return "", false, nil
	case string:
		return t, true, nil
	case bool:
		// Never rely on presence/absence for booleans. Removing an attribute
		// runs coerce(null, default), which yields false even when the default
		// is true, so the value is always written out explicitly.
		return strconv.FormatBool(t), true, nil
	case int:
		return encodeInt(int64(t))
	case int8:
		return encodeInt(int64(t))
	case int16:
		return encodeInt(int64(t))
	case int32:
		return encodeInt(int64(t))
	case int64:
		return encodeInt(t)
	case uint:
		return encodeUint(uint64(t))
	case uint8:
		return encodeUint(uint64(t))
	case uint16:
		return encodeUint(uint64(t))
	case uint32:
		return encodeUint(uint64(t))
	case uint64:
		return encodeUint(t)
	case float32:
		return encodeFloat(float64(t), 32)
	case float64:
		return encodeFloat(t, 64)
	case time.Duration:
		// Milliseconds is the unit JavaScript actually works in.
		return encodeInt(t.Milliseconds())
	case time.Time:
		return t.Format(time.RFC3339Nano), true, nil
	case json.Number:
		return t.String(), true, nil
	case json.RawMessage:
		return string(t), true, nil
	}

	// Interfaces, in the order that gives the most faithful result.
	switch t := v.(type) {
	case json.Marshaler:
		if isNilValue(v) {
			return "", false, nil
		}
		b, err := t.MarshalJSON()
		if err != nil {
			return "", false, fmt.Errorf("alacris: encoding prop: %w", err)
		}
		return string(b), true, nil
	case fmt.Stringer:
		if isNilValue(v) {
			return "", false, nil
		}
		return t.String(), true, nil
	case encoding.TextMarshaler:
		if isNilValue(v) {
			return "", false, nil
		}
		b, err := t.MarshalText()
		if err != nil {
			return "", false, fmt.Errorf("alacris: encoding prop: %w", err)
		}
		return string(b), true, nil
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return "", false, nil
		}
		return EncodeProp(rv.Elem().Interface())
	case reflect.Slice, reflect.Map:
		if rv.IsNil() {
			return "", false, nil
		}
	case reflect.String:
		return rv.String(), true, nil
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool()), true, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return encodeInt(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return encodeUint(rv.Uint())
	case reflect.Float32:
		// rv.Float widens to float64, so formatting at 64 bits would print the
		// widening rather than the value: 0.1 as 0.10000000149011612.
		return encodeFloat(rv.Float(), 32)
	case reflect.Float64:
		return encodeFloat(rv.Float(), 64)
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return "", false, fmt.Errorf("alacris: cannot encode a %s as a prop", rv.Kind())
	}

	b, err := json.Marshal(v)
	if err != nil {
		return "", false, fmt.Errorf("alacris: encoding prop: %w", err)
	}
	return string(b), true, nil
}

// IsZero reports whether v is its type's zero value. Generated wrappers use it
// to decide whether a prop of a named or struct type was set at all, where
// there is no operator they can rely on.
func IsZero(v any) bool {
	if v == nil {
		return true
	}
	return reflect.ValueOf(v).IsZero()
}

func isNilValue(v any) bool {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
		return rv.IsNil()
	}
	return false
}

func encodeInt(n int64) (string, bool, error) {
	if n > MaxSafeInteger || n < -MaxSafeInteger {
		return "", false, fmt.Errorf("%w: %d", ErrUnsafeInteger, n)
	}
	return strconv.FormatInt(n, 10), true, nil
}

func encodeUint(n uint64) (string, bool, error) {
	if n > MaxSafeInteger {
		return "", false, fmt.Errorf("%w: %d", ErrUnsafeInteger, n)
	}
	return strconv.FormatUint(n, 10), true, nil
}

func encodeFloat(f float64, bits int) (string, bool, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "", false, fmt.Errorf("%w: %v", ErrNonFiniteFloat, f)
	}
	// 'g' keeps the shortest round-tripping form and stays inside the grammar
	// JavaScript's unary + accepts, exponents included.
	return strconv.FormatFloat(f, 'g', -1, bits), true, nil
}

// EncodeAttr renders v as an ordinary HTML attribute value, following HTML
// rules rather than alacris' — a true bool becomes a bare attribute and a
// false one disappears. bare is true when the attribute should be written
// without a value.
func EncodeAttr(v any) (s string, bare bool, ok bool, err error) {
	switch t := v.(type) {
	case nil:
		return "", false, false, nil
	case bool:
		return "", t, t, nil
	case *bool:
		if t == nil {
			return "", false, false, nil
		}
		return "", *t, *t, nil
	case string:
		return t, false, true, nil
	}
	s, ok, err = EncodeProp(v)
	return s, false, ok, err
}
