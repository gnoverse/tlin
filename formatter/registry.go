package formatter

import (
	"fmt"
	"sync"
)

// Registry maps rule names to their issueFormatter. Production code
// uses the package-level default via the top-level Register and Get;
// tests can build an isolated Registry with NewRegistry.
type Registry struct {
	mu         sync.RWMutex
	formatters map[string]issueFormatter
}

// NewRegistry returns a fresh empty Registry. Tests use this when
// they need isolation from the package-level default.
func NewRegistry() *Registry {
	return &Registry{formatters: map[string]issueFormatter{}}
}

// Register attaches f to ruleName. Panics on empty name or duplicate
// registration so misregistrations surface at process start (this is
// typically called from init()).
func (r *Registry) Register(ruleName string, f issueFormatter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ruleName == "" {
		panic("formatter.Register: empty rule name")
	}
	if _, dup := r.formatters[ruleName]; dup {
		panic(fmt.Sprintf("formatter.Register: duplicate rule %q", ruleName))
	}
	r.formatters[ruleName] = f
}

// Get returns the formatter registered for ruleName, or
// &GeneralIssueFormatter{} when no specific formatter has been
// registered. Callers always receive a usable formatter.
func (r *Registry) Get(ruleName string) issueFormatter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if f, ok := r.formatters[ruleName]; ok {
		return f
	}
	return &GeneralIssueFormatter{}
}

var defaultRegistry = NewRegistry()

// Register attaches f to ruleName in the package-level default
// registry. Intended for use in init() blocks.
func Register(ruleName string, f issueFormatter) {
	defaultRegistry.Register(ruleName, f)
}

// Get returns the formatter for ruleName from the default registry,
// falling back to GeneralIssueFormatter when no rule-specific
// formatter has registered.
func Get(ruleName string) issueFormatter {
	return defaultRegistry.Get(ruleName)
}
