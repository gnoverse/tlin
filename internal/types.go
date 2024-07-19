package internal

import "go/token"

// SourceCode stores the content of a source code file.
type SourceCode struct {
	Lines []string
}

// Issue represents a lint issue found in the code base.
type Issue struct {
	Rule     string
	Filename string
	Start    token.Position
	End      token.Position
	Message  string
	Suggestion string
}
