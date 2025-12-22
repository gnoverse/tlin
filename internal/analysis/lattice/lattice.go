package lattice

// ValueKind models the zero-ness lattice for integer-like values.
type ValueKind int

const (
	Bottom ValueKind = iota // unreachable
	Zero
	NonZero
	MaybeZero
	Top
)

func (v ValueKind) String() string {
	switch v {
	case Bottom:
		return "Bottom"
	case Zero:
		return "Zero"
	case NonZero:
		return "NonZero"
	case MaybeZero:
		return "MaybeZero"
	case Top:
		return "Top"
	default:
		return "Unknown"
	}
}

// Join returns the least upper bound in the lattice.
func Join(a, b ValueKind) ValueKind {
	if a == Bottom {
		return b
	}
	if b == Bottom {
		return a
	}
	if a == Top || b == Top {
		return Top
	}
	if a == MaybeZero || b == MaybeZero {
		return MaybeZero
	}
	if a == b {
		return a
	}
	// Zero + NonZero.
	return MaybeZero
}

// Meet returns the greatest lower bound in the lattice.
func Meet(a, b ValueKind) ValueKind {
	if a == Bottom || b == Bottom {
		return Bottom
	}
	if a == Top {
		return b
	}
	if b == Top {
		return a
	}
	if a == b {
		return a
	}
	if a == MaybeZero && (b == Zero || b == NonZero) {
		return b
	}
	if b == MaybeZero && (a == Zero || a == NonZero) {
		return a
	}
	return Bottom
}

// AbstractState maps variable names to their zero-ness.
// Missing entries are interpreted as Top.
type AbstractState map[string]ValueKind

// GetValue returns the stored value or Top when absent.
// A nil state represents Bottom (unreachable).
func GetValue(state AbstractState, name string) ValueKind {
	if state == nil {
		return Bottom
	}
	if val, ok := state[name]; ok {
		return val
	}
	return Top
}

// SetValue sets the entry or removes it when value is Top.
func SetValue(state AbstractState, name string, value ValueKind) {
	if state == nil {
		return
	}
	if value == Top {
		delete(state, name)
		return
	}
	state[name] = value
}

// CloneState returns a shallow copy of the abstract state.
func CloneState(state AbstractState) AbstractState {
	if state == nil {
		return nil
	}
	out := make(AbstractState, len(state))
	for k, v := range state {
		out[k] = v
	}
	return out
}

// JoinStates merges two abstract states using Join on each variable.
func JoinStates(a, b AbstractState) AbstractState {
	if a == nil {
		return CloneState(b)
	}
	if b == nil {
		return CloneState(a)
	}
	out := make(AbstractState)
	for name := range a {
		joined := Join(GetValue(a, name), GetValue(b, name))
		SetValue(out, name, joined)
	}
	for name := range b {
		if _, ok := a[name]; ok {
			continue
		}
		joined := Join(GetValue(a, name), GetValue(b, name))
		SetValue(out, name, joined)
	}
	return out
}

// StateEqual reports whether two abstract states are identical.
func StateEqual(a, b AbstractState) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
