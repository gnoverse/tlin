package types

import (
	"encoding/json"
	"fmt"
	"go/token"
)

// Issue represents a lint issue found in the code base.
type Issue struct {
	Rule       string         `json:"rule"`
	Category   string         `json:"category"`
	Filename   string         `json:"filename"`
	Message    string         `json:"message"`
	Suggestion string         `json:"suggestion"`
	Note       string         `json:"note"`
	Start      token.Position `json:"start"`
	End        token.Position `json:"end"`
	Confidence float64        `json:"confidence"` // 0.0 to 1.0
}

func (i Issue) String() string {
	return fmt.Sprintf(
		"rule: %s, filename: %s, message: %s, start: %s, end: %s, confidence: %.2f",
		i.Rule, i.Filename, i.Message, i.Start, i.End, i.Confidence)
}

// PositionWithoutFilename represents a position in the code base without the filename to simplify json marsheling.
type PositionWithoutFilename struct {
	Offset int `json:"offset"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

type IssueWithoutFilename struct {
	Rule       string                  `json:"rule"`
	Category   string                  `json:"category"`
	Message    string                  `json:"message"`
	Suggestion string                  `json:"suggestion"`
	Note       string                  `json:"note"`
	Start      PositionWithoutFilename `json:"start"`
	End        PositionWithoutFilename `json:"end"`
	Confidence float64                 `json:"confidence"`
}

func (i *Issue) MarshalJSON() ([]byte, error) {
	return json.Marshal(&IssueWithoutFilename{
		Rule:       i.Rule,
		Category:   i.Category,
		Message:    i.Message,
		Suggestion: i.Suggestion,
		Note:       i.Note,
		Start:      PositionWithoutFilename{Offset: i.Start.Offset, Line: i.Start.Line, Column: i.Start.Column},
		End:        PositionWithoutFilename{Offset: i.End.Offset, Line: i.End.Line, Column: i.End.Column},
		Confidence: i.Confidence,
	})
}
