package query

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// TODO: should handle Unicode characters?
// TODO: make thread-safe

// buffer represents a state machine based parser buffer that tracks character transitions
// and accumulates tokens. It maintains internal state for parsing both meta-variables
// and regular text tokens.
type buffer struct {
	data   []byte // Raw input bytes
	length int    // Length of input data
	index  int    // Current position in data

	last  States  // Previous state
	state States  // Current state
	class Classes // Character class of current byte

	tokenStart int             // Starting position of current token
	tokenValue strings.Builder // Accumulates characters for current token
}

// newBuffer creates a new buffer instance initialized with the input string.
// The buffer starts in the GO (initial) state.
func newBuffer(input string) *buffer {
	return &buffer{
		data:   []byte(input),
		length: len(input),
		index:  0,
		last:   GO,
		state:  GO,
	}
}

// startToken begins accumulating a new token by recording the start position
// and resetting the token value builder. This should be called at the start
// of parsing any new token.
func (b *buffer) startToken() {
	b.tokenStart = b.index
	b.tokenValue.Reset()
}

// getClass determines the character class of the current byte in the buffer.
// Returns `C_OTHER` if beyond buffer bounds.
func (b *buffer) getClass() Classes {
	if b.index >= b.length {
		return C_OTHER
	}
	return getCharacterClass(b.data[b.index])
}

// transition performs a state transition based on the current character and state.
// Returns the next state and we can detect any error that occurred during transition.
func (b *buffer) transition() (States, error) {
	if b.index >= b.length {
		return __, io.EOF
	}

	b.class = b.getClass()
	nextState := StateTransitionTable[b.state][b.class]

	// check for error state
	if nextState == ER {
		return ER, fmt.Errorf("invalid syntax at position %d", b.index)
	}

	// update state
	b.last = b.state
	b.state = nextState

	return b.state, nil
}

// parseMetaVariable parses a meta-variable pattern like :[name] or :[name:type]
// and returns the corresponding HoleConfig.
//
// The parsing process:
//  1. Starts with ':' character
//  2. Accumulates characters while tracking state transitions
//  3. Handles closing brackets (CB or QB states)
//  4. Optionally processes quantifiers (*, +, ?)
func (b *buffer) parseMetaVariable() (*HoleConfig, error) {
	b.startToken()

	// check initial state
	if b.index >= b.length || b.data[b.index] != ':' {
		return nil, fmt.Errorf("expected ':' at position %d", b.index)
	}

	for b.index < b.length {
		state, err := b.transition()
		if err != nil {
			return nil, err
		}

		// process current character
		b.tokenValue.WriteByte(b.data[b.index])
		b.index++

		// CB(closing bracket) or QB(double closing bracket) state reached
		if state == CB || state == QB {
			// check if next character is quantifier
			if b.index < b.length && isQuantifier(b.data[b.index]) {
				b.tokenValue.WriteByte(b.data[b.index])
				b.index++
				state = QT
			}

			// create token
			value := b.tokenValue.String()
			config, err := ParseHolePattern(value)
			if err != nil {
				return nil, err
			}
			return config, nil
		}
	}

	return nil, fmt.Errorf("incomplete meta variable at position %d", b.tokenStart)
}

// parseText parses regular text content until a special character or meta-variable
// pattern is encountered. Handles both regular text and whitespace.
//
// The parsing process:
//  1. Accumulates characters while in TX or WS states
//  2. Stops at special characters (CL, OB, DB states)
//  3. Returns accumulated text or error if no text found
func (b *buffer) parseText() (string, error) {
	b.startToken()

	for b.index < b.length {
		state, err := b.transition()
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}

		currentChar := b.data[b.index]

		// stop at special characters or meta-variable start
		if state == CL || state == OB || state == DB {
			break
		}

		// handle whitespace or regular text
		if state == TX || state == WS {
			b.tokenValue.WriteByte(currentChar)
			b.index++
			continue
		}

		break
	}

	if b.tokenValue.Len() == 0 {
		return "", fmt.Errorf("no text found at position %d", b.tokenStart)
	}

	return b.tokenValue.String(), nil
}
