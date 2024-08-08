package branch

const PreserveScope = "preserveScope"

// Args contains arguments common to early-return.
type Args struct {
	PreserveScope bool
}

type Chain struct {
	If Branch
	Else Branch
	HasInitializer bool
	HasPriorNonDeviating bool
	AtBlockEnd bool
}