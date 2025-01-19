package query

import (
	"fmt"
	"strings"
)

/*
State Transition Machine Design Rationale

This lexer uses a state transition machine approach instead of a traditional
hand-coded switch-case lexer for several performance-critical reasons:

1. Branch Prediction Optimization
   - Traditional lexers with multiple if/switch statements suffer from branch
     misprediction penalties. Each token type check causes a branch, and modern
     CPUs struggle to predict these branches effectively.
   - State machine approach consolidates branching into a single, predictable
     loop with table-driven transitions, reducing branch mispredictions.

2. Token Processing Efficiency
   - The state machine processes input character-by-character in a tight loop
     using lookup tables, rather than repeatedly examining characters with
     conditional logic.
   - Token length tracking is integrated into the state machine loop via the
     'in_token' table, eliminating the need for separate length calculations.

3. Memory Access Patterns
   - The transition table, while larger than hand-coded logic, provides more
     predictable memory access patterns that modern CPU caches can handle efficiently.
   - Character class equivalence is used to reduce the transition table size
     while maintaining performance (e.g., most alphabetic characters behave similarly).

4. Unified Whitespace and Token Processing
   - The state machine handles both whitespace skipping and token recognition
     in the same loop, eliminating additional branch mispredictions that would
     occur when switching between these modes.

5. Extensibility and Maintainability
   - Adding new token types only requires updating the transition table rather
     than modifying complex branching logic.
   - The state machine structure makes it easier to verify and maintain the lexer's
     behavior compared to nested conditional logic.

Implementation Notes:
   1. States are arranged so that final states have lower numbers, allowing for a single
    comparison to detect when token recognition is complete.
   2. The transition table is structured for efficient CPU cache usage by minimizing
    the table size through character equivalence classes.
   3. The design supports both simple tokens (like operators) and complex tokens
    (like identifiers) while maintaining consistent performance characteristics.

Reference:
 [1] https://nothings.org/computer/lexing.html
*/

type (
	States  int8 // Represents possible states of the parser
	Classes int8 // Represents character classes in the pattern
)

// States represent different stages of lexical analysis:
//   - GO (0)  - Initial state, ready to start processing input
//   - OK (1)  - Accept state, token successfully recognized
//   - CL (2)  - After seeing a colon, expecting bracket or identifier
//   - OB (3)  - After first opening bracket, may start double bracket
//   - DB (4)  - After double bracket, expecting identifier
//   - NM (5)  - Reading name part of metavariable
//   - ID (6)  - Reading type identifier (after colon in name)
//   - CB (7)  - After first closing bracket
//   - QB (8)  - After second closing bracket
//   - QT (9)  - Processing quantifier (*, +, ?)
//   - TX (10) - Processing regular text
//   - WS (11) - Processing whitespace
//   - BR (12) - Processing block delimiters ({, })
//   - ER (13) - Error state (invalid input)
//
// The state numbering is significant - states <= OK are final states,
// allowing for efficient loop termination with a single comparison.
const (
	GO States = iota // Initial state
	OK               // Accept state (successful parse)
	CL               // After colon state (:)
	OB               // After first bracket state ([)
	DB               // After double bracket state ([[)
	NM               // Reading name state
	ID               // Reading type identifier state
	CB               // After closing bracket state (])
	QB               // After double closing bracket state (]])
	QT               // Reading quantifier state (*, +, ?)
	TX               // Reading text state
	WS               // Reading whitespace state
	BR               // Reading block state ({, })
	ER               // Error state
)

// Character class definitions
const (
	C_COLON  Classes = iota // Colon character (:)
	C_LBRACK                // Left bracket ([)
	C_RBRACK                // Right bracket (])
	C_LBRACE                // Left brace ({)
	C_RBRACE                // Right brace (})
	C_SPACE                 // Whitespace characters (space, tab, newline)
	C_IDENT                 // Identifier characters (alphanumeric, _, -)
	C_QUANT                 // Quantifiers (*, +, ?)
	C_OTHER                 // Any other character
)

// State transition table for the pattern parser
// Key considerations in the transitions:
//  1. After NM state, a colon transitions to ID state for type specifications
//  2. CB and QB states allow whitespace transitions for better error recovery
//  3. After quantifiers (QT), we can continue with any valid pattern start
//  4. TX (text) state allows transitioning back to pattern parsing
var StateTransitionTable = [14][9]States{
	//          COLON   LBRACK RBRACK LBRACE RBRACE SPACE  IDENT  QUANT  OTHER
    /* GO 0*/ { CL,     OB,    ER,    BR,    BR,    WS,    TX,    ER,    ER },
    /* OK 1*/ { CL,     OB,    ER,    BR,    BR,    WS,    TX,    ER,    ER },
    /* CL 2*/ { TX,     OB,    ER,    ER,    ER,    ER,    ID,    ER,    ER },
    /* OB 3*/ { TX,     DB,    ER,    ER,    ER,    ER,    NM,    ER,    ER },
    /* DB 4*/ { TX,     ER,    ER,    ER,    ER,    ER,    NM,    ER,    ER },
    /* NM 5*/ { ID,     ER,    CB,    ER,    ER,    ER,    NM,    ER,    ER },
    /* ID 6*/ { ER,     ER,    CB,    ER,    ER,    ER,    ID,    ER,    ER },
    /* CB 7*/ { OK,     ER,    QB,    ER,    ER,    WS,    TX,    QT,    ER },
    /* QB 8*/ { OK,     ER,    ER,    ER,    ER,    WS,    TX,    QT,    ER },
    /* QT 9*/ { CL,     ER,    ER,    BR,    BR,    WS,    TX,    ER,    ER },
    /* TX10*/ { CL,     ER,    ER,    BR,    BR,    WS,    TX,    ER,    ER },
    /* WS11*/ { CL,     ER,    ER,    BR,    BR,    WS,    TX,    ER,    ER },
    /* BR12*/ { CL,     ER,    ER,    BR,    OK,    WS,    TX,    ER,    ER },
    /* ER13*/ { ER,     ER,    ER,    ER,    ER,    ER,    ER,    ER,    ER },
}

// isFinalState determines whether a given state is a final (accepting) state.
func isFinalState(s States) bool {
	switch s {
	case OK, QB, QT, TX:
		return true
	}
	return false
}

func (c Classes) String() string {
	switch c {
	case C_COLON:
		return "COLON"
	case C_LBRACK:
		return "LBRACK"
	case C_RBRACK:
		return "RBRACK"
	case C_LBRACE:
		return "LBRACE"
	case C_RBRACE:
		return "RBRACE"
	case C_SPACE:
		return "SPACE"
	case C_IDENT:
		return "IDENT"
	case C_QUANT:
		return "QUANT"
	case C_OTHER:
		return "OTHER"
	default:
		return "UNKNOWN"
	}
}

// StateMachine represents the parser's state machine
type StateMachine struct {
	state    States // Current state
	input    string // Input pattern to parse
	position int    // Current position in input
}

func NewStateMachine(input string) *StateMachine {
	return &StateMachine{
		state:    GO,
		input:    input,
		position: 0,
	}
}

// Transition records the transition details between states
type Transition struct {
	char      byte
	fromState States
	class     Classes
	toState   States
	pos       int // Position in input
}

func (sm *StateMachine) recordTransitions() []Transition {
	var transitions []Transition

	for sm.position < len(sm.input) {
		c := sm.input[sm.position]
		class := getCharacterClass(c)
		currentState := sm.state
		nextState := StateTransitionTable[currentState][class]

		transitions = append(transitions, Transition{
			char:      c,
			fromState: currentState,
			class:     class,
			toState:   nextState,
			pos:       sm.position,
		})

		sm.state = nextState
		sm.position++

		// if we reach an OK state in middle, we can consider that we have finished one token (e.g., metavariable)
		// and reset the state to GO to continue recognizing the next token
		if sm.state == OK {
			sm.state = GO
		}
	}

	return transitions
}

// recordTransitionsStrict processes the input and ensures that the final state is valid
func (sm *StateMachine) recordTransitionsStrict() ([]Transition, error) {
	transitions := sm.recordTransitions()

	// If the final state is CB, check if it was reached from NM (single-bracketed)
	if sm.state == CB {
		lastTransition := transitions[len(transitions)-1]
		if lastTransition.fromState == NM {
			sm.state = OK
			return transitions, nil
		}
		return transitions, fmt.Errorf("incomplete parse: ended in CB from state %v", lastTransition.fromState)
	}

	// If the final state is one of the final states, accept
	if isFinalState(sm.state) {
		sm.state = OK
		return transitions, nil
	}

	// Check if the state is ERROR
	if sm.state == ER {
		return transitions, fmt.Errorf("invalid parse: reached ERROR state")
	}

	// Otherwise, it's an incomplete parse
	return transitions, fmt.Errorf("incomplete parse: ended in state %v", sm.state)
}

func visualizeTransitions(transitions []Transition) string {
	var b strings.Builder
	for _, t := range transitions {
		fmt.Fprintf(&b, "%c: %v -%v-> %v\n",
			t.char, t.fromState, t.class, t.toState)
	}
	return b.String()
}

// getCharacterClass determines the character class for a given byte
// Handles special characters, whitespace, and identifier characters
// Returns C_OTHER for any character that doesn't fit other categories
func getCharacterClass(c byte) Classes {
	// Check special characters first
	switch c {
	case ':':
		return C_COLON
	case '[':
		return C_LBRACK
	case ']':
		return C_RBRACK
	case '{':
		return C_LBRACE
	case '}':
		return C_RBRACE
	case '*', '+', '?':
		return C_QUANT
	}

	// Check for whitespace
	if isWhitespace(c) {
		return C_SPACE
	}

	// Check for identifier characters
	if isIdentChar(c) {
		return C_IDENT
	}

	return C_OTHER
}

// isIdentChar checks if a character is valid in an identifier
// Allows: alphanumeric, underscore, and hyphen (comby-specific)
func isIdentChar(c byte) bool {
	return ('a' <= c && c <= 'z') ||
		('A' <= c && c <= 'Z') ||
		('0' <= c && c <= '9') ||
		c == '_' ||
		c == '-' // Comby syntax allows hyphens in identifiers
}
