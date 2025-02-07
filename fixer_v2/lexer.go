package fixerv2

import (
	"fmt"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenLiteral
	TokenMeta
)

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenLiteral:
		return "Literal"
	case TokenMeta:
		return "Meta"
	default:
		return "Unknown"
	}
}

type Token struct {
	Type     TokenType
	Value    string
	Ellipsis bool // Used only in `Meta` token
	Line     int
	Col      int
}

func Lex(input string) ([]Token, error) {
	var tokens []Token
	currentLiteral := ""
	line, col := 1, 1
	i := 0

	for i < len(input) {
		c := input[i]

		// Handle escape: e.g., output "\:[" as literal characters
		if c == '\\' {
			if i+1 < len(input) {
				next := input[i+1]
				currentLiteral += string(next)
				if next == '\n' {
					line++
					col = 1
				} else {
					col++
				}
				i += 2
				continue
			} else {
				return nil, fmt.Errorf("line %d col %d: '\\' escape is at the end of input", line, col)
			}
		}

		// Start of metavariable ":["
		if c == ':' && i+1 < len(input) && input[i+1] == '[' {
			// Flush accumulated literal
			if currentLiteral != "" {
				tokens = append(tokens, Token{
					Type:  TokenLiteral,
					Value: currentLiteral,
					Line:  line,
					Col:   col - len(currentLiteral),
				})
				currentLiteral = ""
			}
			i += 2
			col += 2

			// Ignore whitespace inside meta
			for i < len(input) && isWhitespace(input[i]) {
				if input[i] == '\n' {
					line++
					col = 1
				} else {
					col++
				}
				i++
			}
			if i >= len(input) {
				return nil, fmt.Errorf("line %d col %d: metavariable is not terminated", line, col)
			}

			// Parse identifier: must start with alphabet or '_'
			if !isIdentifierStart(input[i]) {
				return nil, fmt.Errorf("line %d col %d: metavariable identifier must start with alphabet or '_'", line, col)
			}
			metaName := ""
			metaName += string(input[i])
			i++
			col++
			for i < len(input) && isIdentifierChar(input[i]) {
				metaName += string(input[i])
				i++
				col++
			}
			// Ignore whitespace between identifier and closing ']'
			for i < len(input) && isWhitespace(input[i]) {
				if input[i] == '\n' {
					line++
					col = 1
				} else {
					col++
				}
				i++
			}
			ellipsis := false
			if i+2 < len(input) && input[i] == '.' && input[i+1] == '.' && input[i+2] == '.' {
				ellipsis = true
				i += 3
				col += 3
				for i < len(input) && isWhitespace(input[i]) {
					if input[i] == '\n' {
						line++
						col = 1
					} else {
						col++
					}
					i++
				}
			}
			if i >= len(input) || input[i] != ']' {
				return nil, fmt.Errorf("line %d col %d: metavariable termination ']' is missing", line, col)
			}
			i++ // Skip ']'
			col++
			tokens = append(tokens, Token{
				Type:     TokenMeta,
				Value:    metaName,
				Ellipsis: ellipsis,
				Line:     line,
				Col:      col,
			})
			continue
		} else {
			currentLiteral += string(c)
			if c == '\n' {
				line++
				col = 1
			} else {
				col++
			}
			i++
		}
	}

	if currentLiteral != "" {
		tokens = append(tokens, Token{
			Type:  TokenLiteral,
			Value: currentLiteral,
			Line:  line,
			Col:   col - len(currentLiteral),
		})
	}
	tokens = append(tokens, Token{
		Type:  TokenEOF,
		Value: "",
		Line:  line,
		Col:   col,
	})
	return tokens, nil
}

func isIdentifierStart(c byte) bool {
	// TODO: add colon for boilerplate. need to update the lexing logic.
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_' || c == ':'
}

func isIdentifierChar(c byte) bool {
	return isIdentifierStart(c) || (c >= '0' && c <= '9')
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
