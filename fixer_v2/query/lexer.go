package query

import (
	"unicode"
)

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

// matchHole checks if the current position indeed indicates a hole
// like :[name] or :[[name]]. If it does, it produces a TokenHole token and
// returns true. Otherwise, it returns false and doesn't modify token list.
func (l *Lexer) matchHole() bool {
	// First, ensure there's enough room for ":[" at least.
	if l.position+1 >= len(l.input) {
		return false
	}
	startPos := l.position

	// Check if the next char is '['
	if l.input[l.position+1] == '[' {
		// If it's "[[", we consider it a "long form": :[[name]]
		isLongForm := (l.position+2 < len(l.input) && l.input[l.position+2] == '[')
		end := l.findHoleEnd(isLongForm)
		if end > 0 {
			// We found a valid closing bracket sequence
			l.addToken(TokenHole, l.input[l.position:end], startPos)
			// Move lexer position so that the main loop will continue from the right place.
			// We do -1 because the main loop increments position once more.
			l.position = end
			return true
		}
	}
	return false
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

// findHoleEnd tries to locate the matching ']' or ']]' depending on whether it's a long form :[[...]].
// Returns the index just AFTER the closing bracket(s), or -1 if no match is found.
func (l *Lexer) findHoleEnd(isLongForm bool) int {
	// If it is a long form :[[ name ]], we look for "]]"
	if isLongForm {
		// Start searching from l.position+3 since we already have ":[["
		for i := l.position + 3; i < len(l.input)-1; i++ {
			if l.input[i] == ']' && l.input[i+1] == ']' {
				return i + 2
			}
		}
	} else {
		// Else, we look for a single ']'
		// Start from l.position+2 because we have ":["
		for i := l.position + 2; i < len(l.input); i++ {
			if l.input[i] == ']' {
				return i + 1
			}
		}
	}
	return -1
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

// extractHoleName extracts the hole name from a string like ":[name]" or ":[[name]]".
// For example, ":[[cond]]" -> "cond", ":[cond]" -> "cond".
// Make sure the token value is well-formed before calling this function.
func extractHoleName(tokenValue string) string {
	// We expect tokenValue to start with :[ or :[[, e.g. :[[cond]]
	if len(tokenValue) > 4 && tokenValue[:3] == ":[[" {
		// :[[ ... ]]
		return tokenValue[3 : len(tokenValue)-2]
	}
	// :[ ... ]
	return tokenValue[2 : len(tokenValue)-1]
}
