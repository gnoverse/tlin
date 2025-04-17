package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/printer"
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
	Severity   Severity       `json:"severity"`
}

func (i Issue) String() string {
	return fmt.Sprintf(
		"rule: %s, filename: %s, message: %s, start: %s, end: %s, severity: %s",
		i.Rule, i.Filename, i.Message, i.Start, i.End, i.Severity)
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
	Severity   Severity                `json:"severity"`
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
		Severity:   i.Severity,
	})
}

type Severity int // Severity of the lint rule

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInfo
	SeverityOff
)

func (s Severity) String() string {
	return [...]string{"ERROR", "WARNING", "INFO", "OFF"}[s]
}

// MarshalJSON marshals the Severity to JSON as a string.
func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s Severity) UnmarshalJSON(data []byte) error {
	var severityStr string
	if err := json.Unmarshal(data, &severityStr); err != nil {
		return err
	}

	switch severityStr {
	case "ERROR":
		s = SeverityError
	case "WARNING":
		s = SeverityWarning
	case "INFO":
		s = SeverityInfo
	case "OFF":
		s = SeverityOff
	default:
		return errors.New("invalid severity level")
	}

	return nil
}

// UnmarshalYAML unmarshals the Severity from YAML as a string.
func (s *Severity) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var severityStr string
	if err := unmarshal(&severityStr); err != nil {
		return err
	}

	switch severityStr {
	case "ERROR":
		*s = SeverityError
	case "WARNING":
		*s = SeverityWarning
	case "INFO":
		*s = SeverityInfo
	case "OFF":
		*s = SeverityOff
	default:
		return errors.New("invalid severity level")
	}

	return nil
}

// Rule represents an individual rule with an ID and severity.
type ConfigRule struct {
	Severity Severity `yaml:"severity"`
	Data     any      `yaml:"data"` // Data can be anything
}

// NodeToString converts an AST node to its string representation
func Node2String(node ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), node)
	return buf.String()
}
