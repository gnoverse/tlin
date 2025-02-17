package fixerv2

import (
	"fmt"
	"strings"
	"unicode"
)

// TODO: refactor lexer after finishing basic implementation
// TODO: add TokenWhiteSpace type to store identation and dedentation
// TODO: add TokenNewLine type to store newlines

// TokenType defines the type of a token
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

// Token represents a lexical token
type Token struct {
	Type     TokenType
	Value    string
	Ellipsis bool // Used only in `Meta` token
	Line     int
	Col      int
}

// Lex performs lexical analysis on the input string
// and returns a sequence of tokens.
func Lex(input string) ([]Token, error) {
	var tokens []Token
	var currentLiteral strings.Builder

	line, col := 1, 1
	i := 0

	flushLiteral := func() {
		if currentLiteral.Len() > 0 {
			tokens = append(tokens, Token{
				Type:  TokenLiteral,
				Value: currentLiteral.String(),
				Line:  line,
				Col:   col - currentLiteral.Len(),
			})
			currentLiteral.Reset()
		}
	}

	for i < len(input) {
		c := input[i]

		// Handle escape characters (e.g., "\:")
		if c == '\\' {
			if i+1 < len(input) {
				next := input[i+1]
				currentLiteral.WriteByte(next)
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
			flushLiteral() // Flush any accumulated literal before processing metavariable
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

			var metaName strings.Builder
			metaName.WriteByte(input[i])
			i++
			col++

			for i < len(input) && isIdentifierChar(input[i]) {
				metaName.WriteByte(input[i])
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

			// Check for ellipsis "..."
			ellipsis := false
			if i+2 < len(input) && input[i] == '.' && input[i+1] == '.' && input[i+2] == '.' {
				ellipsis = true
				i += 3
				col += 3
				// Ignore trailing whitespace
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

			// Ensure closing bracket "]"
			if i >= len(input) || input[i] != ']' {
				return nil, fmt.Errorf("line %d col %d: metavariable termination ']' is missing", line, col)
			}
			i++ // Skip ']'
			col++

			// Append metavariable token
			tokens = append(tokens, Token{
				Type:     TokenMeta,
				Value:    metaName.String(),
				Ellipsis: ellipsis,
				Line:     line,
				Col:      col,
			})
			continue
		}

		// Collect characters for literals
		currentLiteral.WriteByte(c)
		if c == '\n' {
			flushLiteral() // Ensure line information is properly updated
			line++
			col = 1
		} else {
			col++
		}
		i++
	}

	// Flush remaining literal
	flushLiteral()

	// Append EOF token
	tokens = append(tokens, Token{
		Type:  TokenEOF,
		Value: "",
		Line:  line,
		Col:   col,
	})

	return tokens, nil
}

func isIdentifierStart(c byte) bool {
	return unicode.IsLetter(rune(c)) || c == '_'
}

func isIdentifierChar(c byte) bool {
	return isIdentifierStart(c) || isDigit(c)
}

func isWhitespace(c byte) bool {
	return unicode.IsSpace(rune(c))
}

func isDigit(c byte) bool {
	return unicode.IsDigit(rune(c))
}
