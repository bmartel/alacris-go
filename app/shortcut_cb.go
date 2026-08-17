//go:build desktop

package app

import "C"
import "sync"

var (
	hotMu  sync.Mutex
	hotFns = map[int]func(){}
	hotSeq int
	hotKey = map[string]int{}
)

func nextHotID() int {
	hotMu.Lock()
	defer hotMu.Unlock()
	hotSeq++
	return hotSeq
}

func storeHot(id int, keys string, fn func()) {
	hotMu.Lock()
	hotFns[id] = fn
	hotKey[keys] = id
	hotMu.Unlock()
}

func popHot(keys string) (int, bool) {
	hotMu.Lock()
	defer hotMu.Unlock()
	id, ok := hotKey[keys]
	if ok {
		delete(hotKey, keys)
		delete(hotFns, id)
	}
	return id, ok
}

//export goHotkey
func goHotkey(id C.int) {
	hotMu.Lock()
	fn := hotFns[int(id)]
	hotMu.Unlock()
	if fn != nil {
		fn()
	}
}
