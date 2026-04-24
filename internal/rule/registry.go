package rule

import (
	"fmt"
	"sync"
)

// Registry holds rules keyed by their Name(). Use NewTestRegistry for
// tests; the package-level default is used by production code via the
// top-level Register and All functions.
type Registry struct {
	mu    sync.RWMutex
	rules map[string]Rule
}

// NewRegistry returns a fresh empty Registry. Tests use this when
// they need isolation from the package-level default.
func NewRegistry() *Registry {
	return &Registry{rules: map[string]Rule{}}
}

// Register adds r to the registry. Panics if r.Name() is empty or
// already registered. Intended to be called from package init() so
// duplicates surface at process start.
func (r *Registry) Register(rule Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := rule.Name()
	if name == "" {
		panic("rule.Register: empty Name()")
	}
	if _, dup := r.rules[name]; dup {
		panic(fmt.Sprintf("rule.Register: duplicate name %q", name))
	}
	r.rules[name] = rule
}

// All returns a copy of the registered rules. Mutating the returned
// map does not affect the registry.
func (r *Registry) All() map[string]Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]Rule, len(r.rules))
	for k, v := range r.rules {
		out[k] = v
	}
	return out
}

var defaultRegistry = NewRegistry()

// Register adds rule to the package-level default registry. Intended
// for use in init() blocks of rule packages.
func Register(rule Rule) { defaultRegistry.Register(rule) }

// All returns a copy of the default registry's rules.
func All() map[string]Rule { return defaultRegistry.All() }
