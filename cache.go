package why

import (
	"sync"

	"github.com/cespare/xxhash"
	"github.com/d5/tengo/script"
)

type (
	cacheEntry struct {
		base *script.Compiled
		refs *sync.Pool
	}

	scriptSetupFunc func(sc *script.Script)

	scriptCache struct {
		mtx         sync.RWMutex
		cache       map[uint64]cacheEntry
		setupScript scriptSetupFunc
	}
)

func newCache(setupFunc scriptSetupFunc) *scriptCache {
	return &scriptCache{
		cache:       map[uint64]cacheEntry{},
		setupScript: setupFunc,
	}
}

func (sc *scriptCache) get(scriptSource []byte) (uint64, *script.Compiled, error) {
	// Calculate a uint64 hash of the source. Map access with integers
	// is faster than with strings so we use a hash algorithm that outputs
	// uint64 values.
	hashSum := xxhash.Sum64(scriptSource)

	// Check if script is cached by looking up the hash
	sc.mtx.RLock()
	entry, ok := sc.cache[hashSum]
	if ok {
		// Script is cached and we can return a clone of the compiled script.
		defer sc.mtx.RUnlock()
		return hashSum, entry.refs.Get().(*script.Compiled), nil
	}
	sc.mtx.RUnlock()

	// Script was never compiled before.
	sc.mtx.Lock()
	defer sc.mtx.Unlock()

	// Create script and setup all the variables, imports etc.
	s := script.New(scriptSource)
	sc.setupScript(s)

	// Compile the script and check for any errors.
	compiled, err := s.Compile()
	if err != nil {
		return 0, nil, err
	}

	// Create a pool that will clone the compiled script to create
	// new instances.
	refs := &sync.Pool{
		New: func() interface{} {
			return compiled.Clone()
		},
	}

	// Set the cache entry.
	sc.cache[hashSum] = cacheEntry{
		base: compiled,
		refs: refs,
	}

	return hashSum, refs.Get().(*script.Compiled), nil
}

func (sc *scriptCache) put(hashSum uint64, compiled *script.Compiled) {
	sc.mtx.RLock()
	sc.cache[hashSum].refs.Put(compiled)
	sc.mtx.RUnlock()
}
