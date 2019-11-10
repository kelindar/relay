package relay

import (
	"context"
	"encoding/xml"
	"strconv"
	"time"

	"github.com/asafschers/goscore"
	lru "github.com/hashicorp/golang-lru"
	"github.com/kelindar/loader"
	"github.com/spaolacci/murmur3"
	lua "github.com/yuin/gopher-lua"
)

type parseFunc = func([]byte) (interface{}, error)

// Module represents the relay module providing an API to be used for the relay.
type module struct {
	store  *lru.ARCCache
	loader *loader.Loader
}

func newModule() *module {
	cache, err := lru.NewARC(100)
	if err != nil {
		panic(err)
	}

	return &module{
		store:  cache,
		loader: loader.New(),
	}
}

func (m *module) loadModule(state *lua.LState) int {
	// register functions to the table
	mod := state.SetFuncs(state.NewTable(), map[string]lua.LGFunction{
		"hash64": m.hash64,
		"tree":   m.tree,
	})

	// register other stuff
	state.SetField(mod, "version", lua.LString("1.0.0"))

	// returns the module
	state.Push(mod)
	return 1
}

func (m *module) hash64(state *lua.LState) int {
	if state.GetTop() == 0 {
		state.RaiseError("hash64 takes a string argument")
		return 1
	}

	h := murmur3.Sum64([]byte(state.CheckString(1)))
	state.Push(lua.LNumber(h))
	return 1
}

// Score evaluates a PMML-encoded model
func (m *module) tree(state *lua.LState) int {
	if state.GetTop() != 2 {
		state.RaiseError("tree() must have 2 arguments")
		return 1
	}

	// Parse the input features
	v, err := m.load(state.CheckString(1), func(b []byte) (interface{}, error) {
		var model goscore.Node
		err := xml.Unmarshal(b, &model)
		return model, err
	})
	if err != nil {
		state.RaiseError("tree: %s", err.Error())
		return 1
	}

	if node, ok := v.(goscore.Node); ok {
		input := parseTable(state.CheckTable(2))
		score, err := node.TraverseTree(input)
		if err != nil {
			state.RaiseError("tree: %s", err.Error())
			return 1
		}

		state.Push(lua.LNumber(score))
		return 1
	}

	state.RaiseError("tree: invalid node")
	return 1
}

func (m *module) load(uri string, parse parseFunc) (interface{}, error) {
	const timeout = 60 * time.Second
	if v, ok := m.store.Get(uri); ok {
		return v, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	b, err := m.loader.Load(ctx, uri)
	if err != nil {
		return nil, err
	}

	v, err := parse(b)
	if err != nil {
		return nil, err
	}

	m.store.Add(uri, v)
	return v, err
}

// parseTable parses a LUA table to a map[string] -> any
func parseTable(args *lua.LTable) map[string]interface{} {
	input := make(map[string]interface{}, args.Len())
	args.ForEach(func(l lua.LValue, r lua.LValue) {
		if l.Type() != 3 {
			return
		}

		switch r.Type() {
		case 2:
			if v, err := strconv.ParseFloat(r.String(), 64); err == nil {
				input[l.String()] = v
			}
		case 3:
			input[l.String()] = r.String()
		}
	})
	return input
}
