package migration

import (
	"go/token"
)

type Confidence string

const (
	Safe   Confidence = "safe"
	Review Confidence = "review"
	Manual Confidence = "manual"
)

type Edit struct {
	Start      token.Pos  `json:"-"`
	End        token.Pos  `json:"-"`
	NewText    string     `json:"new_text"`
	Category   string     `json:"category"`
	Confidence Confidence `json:"confidence"`
	Rationale  string     `json:"rationale"`
	Position   Position   `json:"position"`
}

type Finding struct {
	Category   string     `json:"category"`
	Confidence Confidence `json:"confidence"`
	Position   Position   `json:"position"`
	Message    string     `json:"message"`
	Suggestion string     `json:"suggestion"`
}

type Position struct {
	Filename string `json:"filename"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

func NewPosition(pos token.Position) Position {
	return Position{
		Filename: pos.Filename,
		Line:     pos.Line,
		Column:   pos.Column,
	}
}
