package live

import (
	"bytes"
	"context"
	"fmt"

	"github.com/a-h/templ"
)

// Op is the kind of change a Patch describes.
type Op string

const (
	// OpProp writes a DOM property: el[key] = value.
	//
	// This is the one that matters. An alacris prop is a signal, so a property
	// write updates exactly the bindings that read it, and nothing else on the
	// page is touched.
	OpProp Op = "p"

	// OpAttr writes an attribute, or removes it when the value is nil. Use it
	// for ordinary HTML attributes; props should go through OpProp.
	OpAttr Op = "a"

	// OpHTML replaces the light-DOM children assigned to one slot.
	OpHTML Op = "h"

	// OpClass toggles a class name.
	OpClass Op = "c"

	// OpReload tells the page to reload itself.
	OpReload Op = "x"
)

// A Patch is one change to one element.
//
// The field names on the wire are single letters because a busy page sends a
// lot of these and every one carries the same four keys.
type Patch struct {
	Op Op `json:"o"`

	// ID is the id attribute of the target element. Empty targets the
	// document element, which only OpReload has any use for.
	ID string `json:"i,omitempty"`

	// Key is the property name for OpProp, the attribute name for OpAttr, the
	// slot name for OpHTML, or the class name for OpClass.
	//
	// Deliberately not omitempty: the default slot's name is "", and omitting
	// it would make an OpHTML patch for the default slot indistinguishable
	// from one with no slot at all.
	Key string `json:"k"`

	Value any `json:"v,omitempty"`
}

// Prop writes a component prop.
//
// name is the JavaScript prop name from define() — "maxCount", not
// "max-count". Server-rendered props go through the attribute, whose name is
// kebab-cased; a live patch writes the DOM property, which keeps the name the
// component declared.
func Prop(id, name string, value any) Patch {
	return Patch{Op: OpProp, ID: id, Key: name, Value: value}
}

// Attr writes an ordinary attribute. A nil value removes it.
func Attr(id, name string, value any) Patch {
	return Patch{Op: OpAttr, ID: id, Key: name, Value: value}
}

// Class turns a class name on or off.
func Class(id, name string, on bool) Patch {
	return Patch{Op: OpClass, ID: id, Key: name, Value: on}
}

// Reload asks the page to reload. It is the escape hatch for a change too
// structural to express as props — a schema migration, a deploy — not an
// everyday update.
func Reload() Patch {
	return Patch{Op: OpReload}
}

// HTML replaces the children assigned to one slot of an element with rendered
// markup. Pass an empty slot name for the default slot.
//
// Props are the better tool almost every time: they are smaller, they do not
// touch the DOM the user is interacting with, and they keep rendering where
// the component put it. Reach for this when the server genuinely owns the
// markup — a rendered document, a chunk of a report.
func HTML(ctx context.Context, id, slot string, c templ.Component) (Patch, error) {
	var b bytes.Buffer
	if c != nil {
		if err := c.Render(templ.InitializeContext(ctx), &b); err != nil {
			return Patch{}, fmt.Errorf("live: rendering slot %q of %q: %w", slot, id, err)
		}
	}
	return Patch{Op: OpHTML, ID: id, Key: slot, Value: b.String()}, nil
}
