package types

import (
	"fmt"
)

// Issue represents a lint issue found in the code base.
type Issue struct {
	Rule       string
	Category   string
	Filename   string
	Message    string
	Suggestion string
	Note       string
	// Start      token.Position
	// End        token.Position
	Start      UniversalPosition
	End        UniversalPosition
	Confidence float64 // 0.0 to 1.0
}

func (i Issue) String() string {
	return fmt.Sprintf(
		"rule: %s, filename: %s, message: %s, start: %d, end: %d, confidence: %.2f",
		i.Rule, i.Filename, i.Message, i.Start.Line, i.End.Line, i.Confidence)
}

type UniversalPosition struct {
	Filename string
	Line     int
	Column   int
	Offset   int
	Length   int
}
