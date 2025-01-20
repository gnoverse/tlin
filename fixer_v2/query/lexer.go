package query

import (
	"unicode"
)

//! TODO: remove this file.

// Lexer is responsible for scanning the input string and producing tokens.
type Lexer struct {
	input    string // the entire input to tokenize
	position int    // current reading position in input
	tokens   []Token
}

// NewLexer returns a new Lexer with the given input and initializes state.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:    input,
		position: 0,
		tokens:   make([]Token, 0),
	}
}

// Tokenize processes the entire input and produces the list of tokens.
func (l *Lexer) Tokenize() []Token {
	for l.position < len(l.input) {
		currentPos := l.position
		switch c := l.input[l.position]; {
		// This might indicate a placeholder like :[name] or :[[name]]
		case c == ':':
			if l.matchHole() {
				// If matchHole returns true, we found :[something] or :[[something]],
				// and the position has been updated. Skip the default token creation.
				continue
			}
			// If matchHole fails, we treat ':' as just a regular text token.
			l.addToken(TokenText, string(c), currentPos)
			l.position++

		case c == '{':
			l.addToken(TokenLBrace, "{", currentPos)
			l.position++

		case c == '}':
			l.addToken(TokenRBrace, "}", currentPos)
			l.position++

		case isWhitespace(c):
			l.lexWhitespace(currentPos)
			l.position++

		default:
			// position incrementing is handled inside `lexText`
			l.lexText(currentPos)
		}
	}

	// At the end, add an EOF token to indicate we're done.
	l.addToken(TokenEOF, "", l.position)
	return l.tokens
}

// lexWhitespace scans consecutive whitespace and produces one TokenWhitespace.
func (l *Lexer) lexWhitespace(startPos int) {
	start := l.position
	for l.position < len(l.input) && isWhitespace(l.input[l.position]) {
		l.position++
	}
	// The substring from start..l.position is all whitespace
	l.addToken(TokenWhitespace, l.input[start:l.position], startPos)
	// Move back one so that the main loop can increment it again.
	l.position--
}

// lexText scans consecutive non-special, non-whitespace characters to produce TokenText.
func (l *Lexer) lexText(startPos int) {
	start := l.position
	for l.position < len(l.input) {
		c := l.input[l.position]

		// starting with `:[`?
		if c == ':' && l.position+1 < len(l.input) && l.input[l.position+1] == '[' {
			break
		}

		if c == '{' || c == '}' || isWhitespace(c) {
			break
		}

		l.position++
	}

	if l.position > start {
		l.addToken(TokenText, l.input[start:l.position], startPos)
	}
}

// addToken is a helper to append a new token to the lexer's token list.
func (l *Lexer) addToken(tokenType TokenType, value string, pos int) {
	l.tokens = append(l.tokens, Token{
		Type:     tokenType,
		Value:    value,
		Position: pos,
	})
}

// isWhitespace checks if the given byte is a space, tab, newline, etc. using unicode.IsSpace.
func isWhitespace(c byte) bool {
	return unicode.IsSpace(rune(c))
}
