package relay

import (
	"bufio"
	"context"
	"io"
	proc "runtime"
	"sync"

	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

// runtime represents a thread-safe code executor.
type runtime struct {
	sync.RWMutex
	mod  *module            // The module providing the API
	pool []*lua.LState      // The pool of states for concurrent use
	code *lua.FunctionProto // The shared code to run
}

// newRuntime creates a new virtual machine that abstracts the state management.
func newRuntime() *runtime {
	return &runtime{
		pool: make([]*lua.LState, 0, proc.NumCPU()),
		mod:  newModule(),
	}
}

// Run runs a script
func (vm *runtime) Run(ctx context.Context) error {
	state := vm.get()
	defer vm.release(state)

	state.SetContext(ctx)

	// Clear the state
	if state.GetTop() != 0 {
		state.Pop(state.GetTop())
	}

	// Make sure the most recent code is present in the state
	lfunc := state.NewFunctionFromProto(vm.code)
	state.Push(lfunc)

	return state.PCall(0, lua.MultRet, nil)
}

// Update updates the code to run
func (vm *runtime) Update(reader io.Reader) error {
	f, err := vm.compile(reader)
	if err != nil {
		return err
	}

	vm.Lock()
	defer vm.Unlock()
	vm.code = f
	return nil
}

// Compile compiles a script into a function that can be shared.
func (vm *runtime) compile(r io.Reader) (*lua.FunctionProto, error) {
	const name = "relay.lua"
	reader := bufio.NewReader(r)
	chunk, err := parse.Parse(reader, name)
	if err != nil {
		return nil, err
	}

	// Compile into a function
	proto, err := lua.Compile(chunk, name)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

// Create creates a new LUA state
func (vm *runtime) create() *lua.LState {
	state := lua.NewState(lua.Options{
		RegistrySize:        1024 * 20, // this is the initial size of the registry
		RegistryMaxSize:     1024 * 80, // this is the maximum size that the registry can grow to. If set to `0` (the default) then the registry will not auto grow
		RegistryGrowStep:    32,        // this is how much to step up the registry by each time it runs out of space. The default is `32`.
		CallStackSize:       120,       // this is the maximum callstack size of this LState
		MinimizeStackMemory: true,      // Defaults to `false` if not specified. If set, the callstack will auto grow and shrink as needed up to a max of `CallStackSize`. If not set, the callstack will be fixed at `CallStackSize`.
	})
	state.PreloadModule("relay", vm.mod.loadModule)
	return state
}

// Get retrieves a state from the pool
func (vm *runtime) get() *lua.LState {
	vm.Lock()
	defer vm.Unlock()

	n := len(vm.pool)
	if n == 0 {
		return vm.create()
	}

	state := vm.pool[n-1]
	vm.pool = vm.pool[0 : n-1]
	return state
}

// Release releases a state back to the pool
func (vm *runtime) release(state *lua.LState) {
	vm.Lock()
	defer vm.Unlock()
	vm.pool = append(vm.pool, state)
}

// Close closes the vm and cleanly disposes of its resources.
func (vm *runtime) Close() error {
	for _, vm := range vm.pool {
		vm.Close()
	}
	return nil
}
