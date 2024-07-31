package types

import "go/token"

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
