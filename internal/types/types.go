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
}

func (i Issue) String() string {
	return fmt.Sprintf(
		"Rule: %s\nCategory: %s\nFilename: %s\nMessage: %s\nSuggestion: %s\nNote: %s\nStart: %s\nEnd: %s\n",
		i.Rule, i.Category, i.Filename, i.Message, i.Suggestion, i.Note, i.Start, i.End,
	)
}
