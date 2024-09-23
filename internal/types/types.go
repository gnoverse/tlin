package types

import (
	"fmt"
	"go/token"
)

// Issue represents a lint issue found in the code base.
type Issue struct {
	Rule       string
	Category   string
	Filename   string
	Message    string
	Suggestion string
	Note       string
	Start      token.Position
	End        token.Position
	Confidence float64 // 0.0 to 1.0
}

func (i Issue) String() string {
	return fmt.Sprintf(
		"rule: %s, filename: %s, message: %s, start: %s, end: %s, confidence: %.2f",
		i.Rule, i.Filename, i.Message, i.Start, i.End, i.Confidence)
}

// IssueEssential represents a simplified version of Issue meant to be used for JSON output.
type IssueEssential struct {
	Rule       string
	Category   string
	Message    string
	Suggestion string
	Note       string
	Snippet    string
	Confidence float64 // 0.0 to 1.0
}
