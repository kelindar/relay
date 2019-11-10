package relay

import (
	"context"
	"time"
)

// Relay represents the instance which wraps the PMML and exposes GetVariable and Track functions.
type Relay struct {
	vm *runtime
}

// New creates a new assembly runtime.
func New() (*Relay, error) {
	return &Relay{
		vm: newRuntime(),
	}, nil
}

// Get retrieves a value of a variable.
func (r *Relay) Get(ctx context.Context) {
	const timeout = 1000 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := r.vm.Run(ctx); err != nil {
		println(err.Error())
	}
}

// Emit emits an event.
func (r *Relay) Emit(ctx context.Context) {

}

// Close closes the relay and cleanly disposes of its resources.
func (r *Relay) Close() error {
	return r.vm.Close()
}
