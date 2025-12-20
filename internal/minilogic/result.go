package minilogic

import "fmt"

// ResultKind represents the kind of execution result.
type ResultKind int

const (
	// ResultContinue indicates normal execution continues.
	ResultContinue ResultKind = iota
	// ResultReturn indicates a return statement was executed.
	ResultReturn
	// ResultBreak indicates a break statement was executed.
	ResultBreak
	// ResultContinueLoop indicates a continue statement was executed.
	ResultContinueLoop
	// ResultUnknown indicates the result cannot be determined.
	ResultUnknown
)

func (k ResultKind) String() string {
	switch k {
	case ResultContinue:
		return "Continue"
	case ResultReturn:
		return "Return"
	case ResultBreak:
		return "Break"
	case ResultContinueLoop:
		return "ContinueLoop"
	case ResultUnknown:
		return "Unknown"
	default:
		return "?"
	}
}

// Result represents the outcome of evaluating a statement.
// Based on MiniLogic v2.1 spec:
// Result = Continue(Env) | Return(Value?) | Break | ContinueLoop | Unknown
type Result struct {
	Kind  ResultKind
	Env   *Env         // valid for Continue
	Value Value        // valid for Return (can be nil)
	Calls []CallRecord // tracked function calls (for OpaqueCalls policy)
}

// CallRecord represents a function call that was executed.
type CallRecord struct {
	Func string
	Args []Value
}

func (c CallRecord) String() string {
	result := c.Func + "("
	for i, arg := range c.Args {
		if i > 0 {
			result += ", "
		}
		result += arg.String()
	}
	return result + ")"
}

// ContinueResult creates a Continue result with the given environment.
func ContinueResult(env *Env) Result {
	return Result{Kind: ResultContinue, Env: env}
}

// ContinueResultWithCalls creates a Continue result with tracked calls.
func ContinueResultWithCalls(env *Env, calls []CallRecord) Result {
	return Result{Kind: ResultContinue, Env: env, Calls: calls}
}

// ReturnResult creates a Return result with the given value.
func ReturnResult(val Value, calls []CallRecord) Result {
	return Result{Kind: ResultReturn, Value: val, Calls: calls}
}

// ReturnNilResult creates a Return result with nil value.
func ReturnNilResult(calls []CallRecord) Result {
	return Result{Kind: ResultReturn, Value: NilValue{}, Calls: calls}
}

// BreakResult creates a Break result.
func BreakResult(calls []CallRecord) Result {
	return Result{Kind: ResultBreak, Calls: calls}
}

// ContinueLoopResult creates a ContinueLoop result.
func ContinueLoopResult(calls []CallRecord) Result {
	return Result{Kind: ResultContinueLoop, Calls: calls}
}

// UnknownResult creates an Unknown result.
func UnknownResult() Result {
	return Result{Kind: ResultUnknown}
}

// IsTerminating returns true if this result represents a terminating statement.
func (r Result) IsTerminating() bool {
	return r.Kind == ResultReturn || r.Kind == ResultBreak || r.Kind == ResultContinueLoop
}

// String returns a string representation of the result.
func (r Result) String() string {
	switch r.Kind {
	case ResultContinue:
		return fmt.Sprintf("Continue(%s)", r.Env.String())
	case ResultReturn:
		if r.Value == nil {
			return "Return(nil)"
		}
		return fmt.Sprintf("Return(%s)", r.Value.String())
	case ResultBreak:
		return "Break"
	case ResultContinueLoop:
		return "ContinueLoop"
	case ResultUnknown:
		return "Unknown"
	default:
		return "?"
	}
}

// Equal checks if two results are equivalent.
func (r Result) Equal(other Result) bool {
	if r.Kind != other.Kind {
		return false
	}

	switch r.Kind {
	case ResultContinue:
		// Environments must be equal
		if !r.Env.Equal(other.Env) {
			return false
		}
	case ResultReturn:
		// Returned values must be equal
		if r.Value == nil && other.Value == nil {
			// both nil
		} else if r.Value == nil || other.Value == nil {
			return false
		} else if !r.Value.Equal(other.Value) {
			return false
		}
	case ResultBreak, ResultContinueLoop, ResultUnknown:
		// Kind match is sufficient
	}

	// Check call sequence equality
	if len(r.Calls) != len(other.Calls) {
		return false
	}
	for i := range r.Calls {
		if !callsEqual(r.Calls[i], other.Calls[i]) {
			return false
		}
	}

	return true
}

func callsEqual(a, b CallRecord) bool {
	if a.Func != b.Func {
		return false
	}
	if len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		if !a.Args[i].Equal(b.Args[i]) {
			return false
		}
	}
	return true
}
