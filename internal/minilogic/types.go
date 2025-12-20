package minilogic

import "fmt"

// Value represents a symbolic or concrete value in the MiniLogic system.
// Values can be integers, booleans, strings, nil, or symbolic variables.
type Value interface {
	isValue()
	String() string
	Equal(other Value) bool
}

// IntValue represents an integer constant.
type IntValue struct {
	Val int64
}

func (IntValue) isValue() {}
func (v IntValue) String() string {
	return fmt.Sprintf("%d", v.Val)
}

func (v IntValue) Equal(other Value) bool {
	if o, ok := other.(IntValue); ok {
		return v.Val == o.Val
	}
	return false
}

// BoolValue represents a boolean constant.
type BoolValue struct {
	Val bool
}

func (BoolValue) isValue() {}
func (v BoolValue) String() string {
	return fmt.Sprintf("%t", v.Val)
}

func (v BoolValue) Equal(other Value) bool {
	if o, ok := other.(BoolValue); ok {
		return v.Val == o.Val
	}
	return false
}

// StringValue represents a string constant.
type StringValue struct {
	Val string
}

func (StringValue) isValue() {}
func (v StringValue) String() string {
	return fmt.Sprintf("%q", v.Val)
}

func (v StringValue) Equal(other Value) bool {
	if o, ok := other.(StringValue); ok {
		return v.Val == o.Val
	}
	return false
}

// NilValue represents nil.
type NilValue struct{}

func (NilValue) isValue() {}
func (NilValue) String() string {
	return "nil"
}

func (v NilValue) Equal(other Value) bool {
	_, ok := other.(NilValue)
	return ok
}

// SymbolicValue represents an unknown symbolic value.
// Used when we cannot determine the concrete value statically.
type SymbolicValue struct {
	Name string
}

func (SymbolicValue) isValue() {}
func (v SymbolicValue) String() string {
	return fmt.Sprintf("<%s>", v.Name)
}

func (v SymbolicValue) Equal(other Value) bool {
	if o, ok := other.(SymbolicValue); ok {
		return v.Name == o.Name
	}
	return false
}

// Env represents the symbolic environment as a key-value map.
// It maps variable names to their values.
type Env struct {
	vars   map[string]Value
	parent *Env // for scoped environments (init statements)
}

// NewEnv creates a new empty environment.
func NewEnv() *Env {
	return &Env{
		vars: make(map[string]Value),
	}
}

// NewChildEnv creates a new environment with the given parent.
// Variables in the child shadow those in the parent.
func NewChildEnv(parent *Env) *Env {
	return &Env{
		vars:   make(map[string]Value),
		parent: parent,
	}
}

// Get retrieves the value of a variable.
// Returns nil if the variable is not found.
func (e *Env) Get(name string) Value {
	if v, ok := e.vars[name]; ok {
		return v
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return nil
}

// Set sets the value of a variable in the current scope.
func (e *Env) Set(name string, val Value) {
	e.vars[name] = val
}

// Clone creates a deep copy of the environment.
func (e *Env) Clone() *Env {
	newEnv := &Env{
		vars:   make(map[string]Value, len(e.vars)),
		parent: e.parent, // parent is shared (immutable reference)
	}
	for k, v := range e.vars {
		newEnv.vars[k] = v
	}
	return newEnv
}

// Equal checks if two environments have the same variable bindings.
func (e *Env) Equal(other *Env) bool {
	if e == nil && other == nil {
		return true
	}
	if e == nil || other == nil {
		return false
	}

	// Get all keys from both environments
	allKeys := make(map[string]struct{})
	e.collectKeys(allKeys)
	other.collectKeys(allKeys)

	for k := range allKeys {
		v1 := e.Get(k)
		v2 := other.Get(k)
		if v1 == nil && v2 == nil {
			continue
		}
		if v1 == nil || v2 == nil {
			return false
		}
		if !v1.Equal(v2) {
			return false
		}
	}
	return true
}

func (e *Env) collectKeys(keys map[string]struct{}) {
	for k := range e.vars {
		keys[k] = struct{}{}
	}
	if e.parent != nil {
		e.parent.collectKeys(keys)
	}
}

// String returns a string representation of the environment.
func (e *Env) String() string {
	result := "{"
	first := true
	for k, v := range e.vars {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s: %s", k, v.String())
		first = false
	}
	if e.parent != nil {
		result += " | parent: " + e.parent.String()
	}
	result += "}"
	return result
}

// Keys returns all variable names in this environment (not including parent).
func (e *Env) Keys() []string {
	keys := make([]string, 0, len(e.vars))
	for k := range e.vars {
		keys = append(keys, k)
	}
	return keys
}
