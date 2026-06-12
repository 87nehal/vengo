package autowire

import "sync"

var (
	mu           sync.Mutex
	constructors []any
)

// Register marks a constructor for auto-wiring into the container.
func Register(constructor any) {
	mu.Lock()
	defer mu.Unlock()
	constructors = append(constructors, constructor)
}

// Constructors returns all registered autowire constructors.
func Constructors() []any {
	mu.Lock()
	defer mu.Unlock()
	return append([]any(nil), constructors...)
}
