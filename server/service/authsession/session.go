package authsession

import "sync"

type Closer func(code int, text string)

var registry = struct {
	sync.Mutex
	nextID  int
	closers map[int]Closer
}{closers: make(map[int]Closer)}

func Register(closer Closer) func() {
	registry.Lock()
	registry.nextID++
	id := registry.nextID
	registry.closers[id] = closer
	registry.Unlock()

	return func() {
		registry.Lock()
		delete(registry.closers, id)
		registry.Unlock()
	}
}

func CloseAll(code int, text string) {
	registry.Lock()
	closers := make([]Closer, 0, len(registry.closers))
	for _, closer := range registry.closers {
		closers = append(closers, closer)
	}
	registry.Unlock()

	for _, closer := range closers {
		closer(code, text)
	}
}
