package branch

type BranchKind int

const (
	Empty BranchKind = iota

	// Return branches return from the current function
	Return

	// Continue branches continue a surrounding `for` loop
	Continue

	// Break branches break out of a surrounding `for` loop
	Break

	// Goto branches jump to a label
	Goto

	// Panic panics the current program
	Panic

	// Exit exits the current program
	Exit

	// Regular branches not categorized as any of the above
	Regular
)

func (k BranchKind) IsEmpty() bool  { return k == Empty }
func (k BranchKind) Returns() bool  { return k == Return }
func (k BranchKind) Branch() Branch { return Branch{BranchKind: k} }

func (k BranchKind) Deviates() bool {
	switch k {
	case Empty, Regular:
		return false
	case Return, Continue, Break, Goto, Panic, Exit:
		return true
	default:
		panic("unreachable")
	}
}

func (k BranchKind) String() string {
	switch k {
	case Empty:
		return ""
	case Regular:
		return "..."
	case Return:
		return "... return"
	case Continue:
		return "... continue"
	case Break:
		return "... break"
	case Goto:
		return "... goto"
	case Panic:
		return "... panic()"
	case Exit:
		return "... os.Exit()"
	default:
		panic("invalid kind")
	}
}
