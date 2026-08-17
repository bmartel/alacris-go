// Package model is the example's server-side state.
//
// It is deliberately ordinary Go: a mutex, a slice, and methods. Nothing in it
// knows about alacris, HTTP or the browser. The wiring in main.go is what
// turns a change here into a prop write there.
package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

const (
	ColTodo  = "todo"
	ColDoing = "doing"
	ColDone  = "done"
)

// A Column is one list on the board.
type Column struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// An Item is one card on the shared board.
//
// The JSON tags are the contract with the component: this struct is what ends
// up in the items attribute, and what the component reads in its each() row.
type Item struct {
	ID     int      `json:"id"`
	Text   string   `json:"text"`
	Body   string   `json:"body"`
	Who    []string `json:"who"`
	Labels []string `json:"labels"`
	Column string   `json:"column"`
	Rank   int      `json:"rank"`
}

// ErrEmpty is returned when a card or list has no text.
var ErrEmpty = errors.New("a card needs some text")

// MaxText is the longest title the board accepts. Input arriving over the wire
// is input, whoever it claims to be from.
const MaxText = 200

// MaxBody is the longest description a card will keep.
const MaxBody = 2000

// MaxLabel is the longest slug a label will keep.
const MaxLabel = 24

// MaxLabels is how many labels a single card will keep. The editor always
// offers launch/copy/blocked; cleanLabels slugs anything else rather than
// dropping it.
const MaxLabels = 8

// People the board can assign to a card.
var KnownPeople = []string{"You", "Ada Lovelace", "Ben Linus", "Cara Moss"}

// MaxLists caps how many columns a board can grow to.
const MaxLists = 12

// A List is a set of cards, safe for concurrent use.
type List struct {
	mu      sync.Mutex
	columns []Column
	items   []Item
	nextID  int
	nextCol int
}

// New returns a board already holding a few cards in flight.
func New() *List {
	l := &List{
		columns: []Column{
			{ID: ColTodo, Title: "To do"},
			{ID: ColDoing, Title: "Doing"},
			{ID: ColDone, Title: "Done"},
		},
		nextCol: 3,
	}
	l.must(Item{
		Text: "Cut the hero video", Who: []string{"Ada Lovelace"}, Column: ColDoing,
		Body:   "Fifteen seconds, no voiceover. Cut from the Berlin footage.",
		Labels: []string{"launch"},
	})
	l.must(Item{
		Text: "Pricing page", Who: []string{"Ben Linus"}, Column: ColTodo,
		Body:   "Three tiers, annual toggle, one footnote for the enterprise call.",
		Labels: []string{"copy"},
	})
	l.must(Item{Text: "Onboarding mail", Who: []string{"Cara Moss"}, Column: ColTodo, Labels: []string{"copy"}})
	l.must(Item{Text: "Ship 0.3", Who: []string{"Ada Lovelace"}, Column: ColDone, Labels: []string{"launch"}})
	return l
}

func (l *List) must(it Item) {
	l.nextID++
	it.ID = l.nextID
	if it.Column == "" {
		it.Column = ColTodo
	}
	it.Rank = l.nextRank(it.Column)
	l.items = append(l.items, it)
}

func (l *List) nextRank(column string) int {
	n := 0
	for _, it := range l.items {
		if it.Column == column {
			n++
		}
	}
	return n
}

func (l *List) colIndex(id string) int {
	for i, c := range l.columns {
		if c.ID == id {
			return i
		}
	}
	return len(l.columns)
}

func (l *List) hasColumn(id string) bool {
	return l.colIndex(id) < len(l.columns)
}

// Columns returns a copy of the lists, in board order.
func (l *List) Columns() []Column {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Column(nil), l.columns...)
}

// ColumnIDs returns every list id, for the demo collaborator to pick from.
func (l *List) ColumnIDs() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.columns))
	for i, c := range l.columns {
		out[i] = c.ID
	}
	return out
}

// Items returns a copy of the board, ordered by column then rank.
func (l *List) Items() []Item {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := append([]Item(nil), l.items...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		ra, rb := l.colIndex(a.Column), l.colIndex(b.Column)
		if ra != rb {
			return ra < rb
		}
		if a.Rank != b.Rank {
			return a.Rank < b.Rank
		}
		return a.ID < b.ID
	})
	return out
}

// Members returns each distinct card owner, in the order they first appeared.
func (l *List) Members() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	seen := make(map[string]bool)
	out := make([]string, 0, 8)
	for _, it := range l.items {
		for _, name := range it.Who {
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// Add files a card at the end of a lane and returns it.
func (l *List) Add(text, column string) (Item, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Item{}, ErrEmpty
	}
	if len(text) > MaxText {
		text = text[:MaxText]
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.hasColumn(column) {
		if len(l.columns) == 0 {
			return Item{}, ErrEmpty
		}
		column = l.columns[0].ID
	}
	l.nextID++
	item := Item{ID: l.nextID, Text: text, Who: []string{"You"}, Labels: []string{}, Column: column, Rank: l.nextRank(column)}
	l.items = append(l.items, item)
	return item, nil
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}

func slugLabel(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	hyphen := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			hyphen = false
		default:
			if b.Len() > 0 && !hyphen {
				b.WriteByte('-')
				hyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > MaxLabel {
		out = strings.TrimRight(out[:MaxLabel], "-")
	}
	return out
}

func cleanLabels(in []string) []string {
	out := make([]string, 0, MaxLabels)
	seen := make(map[string]bool, MaxLabels)
	for _, s := range in {
		s = slugLabel(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
		if len(out) >= MaxLabels {
			break
		}
	}
	return out
}

func cleanPeople(in []string) []string {
	allowed := make(map[string]bool, len(KnownPeople))
	for _, n := range KnownPeople {
		allowed[n] = true
	}
	out := make([]string, 0, len(KnownPeople))
	seen := make(map[string]bool, len(KnownPeople))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if !allowed[s] || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// Update rewrites a card's title, description, members and labels. An unknown
// id is ignored. An empty title leaves the existing one alone.
func (l *List) Update(id int, text, body string, who, labels []string) {
	text = clip(text, MaxText)
	body = clip(body, MaxBody)
	who = cleanPeople(who)
	labels = cleanLabels(labels)

	l.mu.Lock()
	defer l.mu.Unlock()
	for i, it := range l.items {
		if it.ID != id {
			continue
		}
		if text != "" {
			it.Text = text
		}
		it.Body = body
		it.Who = who
		it.Labels = labels
		l.items[i] = it
		return
	}
}

// Remove deletes a card. An unknown id is ignored.
func (l *List) Remove(id int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := l.items[:0]
	for _, it := range l.items {
		if it.ID == id {
			continue
		}
		kept = append(kept, it)
	}
	l.items = kept
}

// RemoveColumn deletes a list and every card on it. An unknown id is ignored.
func (l *List) RemoveColumn(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	cols := l.columns[:0]
	for _, c := range l.columns {
		if c.ID == id {
			continue
		}
		cols = append(cols, c)
	}
	if len(cols) == len(l.columns) {
		return
	}
	l.columns = cols
	kept := l.items[:0]
	for _, it := range l.items {
		if it.Column == id {
			continue
		}
		kept = append(kept, it)
	}
	l.items = kept
}

// AddColumn appends a list on the right.
func (l *List) AddColumn(title string) (Column, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Column{}, ErrEmpty
	}
	if len(title) > 80 {
		title = title[:80]
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.columns) >= MaxLists {
		return Column{}, errors.New("too many lists")
	}
	l.nextCol++
	col := Column{ID: fmt.Sprintf("list-%d", l.nextCol), Title: title}
	l.columns = append(l.columns, col)
	return col, nil
}

// RenameColumn sets a list's title. An unknown id or empty title is ignored.
func (l *List) RenameColumn(id, title string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}
	if len(title) > 80 {
		title = title[:80]
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	for i, c := range l.columns {
		if c.ID == id {
			l.columns[i].Title = title
			return
		}
	}
}

// Move places a card in a lane at index. An unknown id is a stale click and
// is ignored. A negative index, or one past the end, appends.
func (l *List) Move(id int, column string, index int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.hasColumn(column) {
		return
	}

	var item Item
	found := false
	kept := make([]Item, 0, len(l.items))
	for _, it := range l.items {
		if it.ID == id {
			item = it
			found = true
			continue
		}
		kept = append(kept, it)
	}
	if !found {
		return
	}
	item.Column = column

	inCol := make([]Item, 0, len(kept)+1)
	others := make([]Item, 0, len(kept))
	for _, it := range kept {
		if it.Column == column {
			inCol = append(inCol, it)
		} else {
			others = append(others, it)
		}
	}
	sort.SliceStable(inCol, func(i, j int) bool {
		if inCol[i].Rank != inCol[j].Rank {
			return inCol[i].Rank < inCol[j].Rank
		}
		return inCol[i].ID < inCol[j].ID
	})
	if index < 0 || index > len(inCol) {
		index = len(inCol)
	}
	inCol = append(inCol[:index:index], append([]Item{item}, inCol[index:]...)...)
	for i := range inCol {
		inCol[i].Rank = i
	}
	l.items = append(others, inCol...)
}

// MoveColumn places a list at index. An unknown id is ignored. A negative
// index, or one past the end, appends.
func (l *List) MoveColumn(id string, index int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	from := -1
	for i, c := range l.columns {
		if c.ID == id {
			from = i
			break
		}
	}
	if from < 0 {
		return
	}
	col := l.columns[from]
	rest := append([]Column(nil), l.columns[:from]...)
	rest = append(rest, l.columns[from+1:]...)
	if index < 0 || index > len(rest) {
		index = len(rest)
	}
	if from == index {
		return
	}
	l.columns = append(rest[:index:index], append([]Column{col}, rest[index:]...)...)
}

// IDs returns every card id, for the demo collaborator to pick from.
func (l *List) IDs() []int {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]int, len(l.items))
	for i, it := range l.items {
		out[i] = it.ID
	}
	return out
}
