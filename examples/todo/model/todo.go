// Package model is the example's server-side state.
//
// It is deliberately ordinary Go: a mutex, a slice, and methods. Nothing in it
// knows about alacris, HTTP or the browser. The wiring in main.go is what
// turns a change here into a prop write there.
package model

import (
	"errors"
	"strings"
	"sync"
)

// An Item is one todo.
//
// The JSON tags are the contract with the component: this struct is what ends
// up in the items attribute, and what the component reads in its each() row.
type Item struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// ErrEmpty is returned when a todo has no text.
var ErrEmpty = errors.New("a todo needs some text")

// MaxText is the longest todo the list accepts. Input arriving over the wire
// is input, whoever it claims to be from.
const MaxText = 200

// A List is a set of todos, safe for concurrent use.
type List struct {
	mu     sync.Mutex
	items  []Item
	nextID int
}

// New returns a list holding the given texts.
func New(texts ...string) *List {
	l := &List{}
	for _, t := range texts {
		_, _ = l.Add(t)
	}
	return l
}

// Items returns a copy of the list, safe to hand to a renderer.
func (l *List) Items() []Item {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Item(nil), l.items...)
}

// Add appends a todo and returns it.
func (l *List) Add(text string) (Item, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Item{}, ErrEmpty
	}
	if len(text) > MaxText {
		text = text[:MaxText]
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.nextID++
	item := Item{ID: l.nextID, Text: text}
	l.items = append(l.items, item)
	return item, nil
}

// Toggle flips one todo's done flag. An id that is not in the list is not an
// error: it is a stale click from a page that has not caught up yet.
func (l *List) Toggle(id int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.items {
		if l.items[i].ID == id {
			l.items[i].Done = !l.items[i].Done
			return
		}
	}
}

// Remove deletes one todo.
func (l *List) Remove(id int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.items {
		if l.items[i].ID == id {
			l.items = append(l.items[:i], l.items[i+1:]...)
			return
		}
	}
}

// Remaining counts the todos that are not done.
func (l *List) Remaining() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, it := range l.items {
		if !it.Done {
			n++
		}
	}
	return n
}
