package query

// Match represents a single pattern match result
type Match struct {
	Start    int               // Start position in source
	End      int               // End position in source
	Bindings map[string]string // Meta-variable bindings
}
