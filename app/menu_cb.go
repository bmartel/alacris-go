//go:build desktop

package app

import "C"
import "sync"

var (
	menuMu  sync.Mutex
	menuFns = map[int]func(){}
	menuSeq int
)

func resetMenuFns() {
	menuMu.Lock()
	menuFns = map[int]func(){}
	menuSeq = 1
	menuMu.Unlock()
}

func registerMenuDo(w *Window, do func(*Window)) int {
	menuMu.Lock()
	defer menuMu.Unlock()
	id := menuSeq
	menuSeq++
	menuFns[id] = func() { do(w) }
	return id
}

//export goMenuInvoke
func goMenuInvoke(id C.int) {
	menuMu.Lock()
	fn := menuFns[int(id)]
	menuMu.Unlock()
	if fn != nil {
		fn()
	}
}
