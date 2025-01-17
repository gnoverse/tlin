package query

import (
	"fmt"
	"strings"
)

// HoleType defines the type of hole pattern
type HoleType int

const (
	HoleAny        HoleType = iota // :[[name]] or :[name]
	HoleIdentifier                 // :[[id:identifier]]
	HoleBlock                      // :[[block:block]]
	HoleWhitespace                 // :[[ws:whitespace]]
	HoleExpression                 // :[[expr:expression]]
)

func (h HoleType) String() string {
	switch h {
	case HoleAny:
		return "any"
	case HoleIdentifier:
		return "identifier"
	case HoleBlock:
		return "block"
	case HoleWhitespace:
		return "whitespace"
	case HoleExpression:
		return "expression"
	default:
		return "unknown"
	}
}

// Quantifier defines repetition patterns
type Quantifier int

const (
	QuantNone       Quantifier = iota // No quantifier (exactly once)
	QuantZeroOrMore                   // * (zero or more times)
	QuantOneOrMore                    // + (one or more times)
	QuantZeroOrOne                    // ? (zero or one time)
)

func (q Quantifier) String() string {
	switch q {
	case QuantNone:
		return ""
	case QuantZeroOrMore:
		return "*"
	case QuantOneOrMore:
		return "+"
	case QuantZeroOrOne:
		return "?"
	default:
		return "unknown"
	}
}

// HoleConfig stores configuration for a hole pattern
type HoleConfig struct {
	Type       HoleType
	Quantifier Quantifier
	Name       string
}

// HoleNode represents a placeholder in the pattern like :[name] or :[[name]].
type HoleNode struct {
	Config HoleConfig
	pos    int
}

func NewHoleNode(name string, pos int) *HoleNode {
	return &HoleNode{
		Config: HoleConfig{
			Name:       name,
			Type:       HoleAny,
			Quantifier: QuantNone,
		},
		pos: pos,
	}
}

func (h *HoleNode) Type() NodeType { return NodeHole }

func (h *HoleNode) String() string {
	if h.Config.Type == HoleAny && h.Config.Quantifier == QuantNone {
		return fmt.Sprintf("HoleNode(%s)", h.Config.Name)
	}
	return fmt.Sprintf("HoleNode(%s:%s)%s", h.Config.Name, h.Config.Type, h.Config.Quantifier)
}

func (h *HoleNode) Position() int { return h.pos }

// ParseHolePattern parses a hole pattern string and returns a HoleConfig
// Format: :[[name:type]] or :[[name:type]]*
func ParseHolePattern(pattern string) (*HoleConfig, error) {
	// Skip : and opening brackets
	start := 1
	if pattern[1] == '[' && pattern[2] == '[' {
		start = 3
	} else if pattern[1] == '[' {
		start = 2
	} else {
		return nil, fmt.Errorf("invalid hole pattern: %s", pattern)
	}

	// Find the end of the pattern
	// Find end excluding quantifier and closing brackets
	end := len(pattern) - 1

	// Check for quantifier
	hasQuantifier := end >= 0 && (pattern[end] == '*' || pattern[end] == '+' || pattern[end] == '?')
	if hasQuantifier {
		end--
	}

	// Remove closing brackets
	if end >= 1 && pattern[end-1:end+1] == "]]" {
		end -= 2
	} else if end >= 0 && pattern[end] == ']' {
		end--
	}

	if end < start {
		return nil, fmt.Errorf("invalid hole pattern: %s", pattern)
	}

	// Parse name and type
	content := pattern[start : end+1]
	parts := strings.Split(content, ":")
	config := &HoleConfig{
		Name:       parts[0],
		Type:       HoleAny,
		Quantifier: QuantNone,
	}

	// Parse type if specified
	if len(parts) > 1 {
		switch parts[1] {
		case "identifier":
			config.Type = HoleIdentifier
		case "block":
			config.Type = HoleBlock
		case "whitespace":
			config.Type = HoleWhitespace
		case "expression":
			config.Type = HoleExpression
		default:
			return nil, fmt.Errorf("unknown hole type: %s", parts[1])
		}
	}

	// Set quantifier if found earlier
	if hasQuantifier {
		switch pattern[len(pattern)-1] {
		case '*':
			config.Quantifier = QuantZeroOrMore
		case '+':
			config.Quantifier = QuantOneOrMore
		case '?':
			config.Quantifier = QuantZeroOrOne
		}
	}

	return config, nil
}

func (l *Lexer) matchHole() bool {
	if l.position+1 >= len(l.input) {
		return false
	}
	startPos := l.position

	if l.input[l.position+1] == '[' {
		isLongForm := (l.position+2 < len(l.input) && l.input[l.position+2] == '[')
		end := l.findHoleEnd(isLongForm)
		if end > 0 {
			// Check for quantifier
			if end < len(l.input) && isQuantifier(l.input[end]) {
				end++
			}

			value := l.input[l.position:end]
			config, err := ParseHolePattern(value)
			if err != nil {
				// If parsing fails, try to extract at least the name and create a basic config
				basicName := extractHoleName(value)
				basicConfig := HoleConfig{
					Name:       basicName,
					Type:       HoleAny,
					Quantifier: QuantNone,
				}
				l.addTokenWithHoleConfig(TokenHole, value, startPos, basicConfig)
			} else {
				// Create a token with the parsed configuration
				l.addTokenWithHoleConfig(TokenHole, value, startPos, *config)
			}
			l.position = end
			return true
		}
	}
	return false
}

func (l *Lexer) addTokenWithHoleConfig(tokenType TokenType, value string, pos int, config HoleConfig) {
	l.tokens = append(l.tokens, Token{
		Type:       tokenType,
		Value:      value,
		Position:   pos,
		HoleConfig: &config,
	})
}

// isQuantifier checks if a character is a valid quantifier
func isQuantifier(c byte) bool {
	return c == '*' || c == '+' || c == '?'
}

func (l *Lexer) findHoleEnd(isLongForm bool) int {
	if isLongForm {
		for i := l.position + 3; i < len(l.input)-1; i++ {
			if l.input[i] == ']' && l.input[i+1] == ']' {
				// Check if there's a quantifier after the closing brackets
				if i+2 < len(l.input) && isQuantifier(l.input[i+2]) {
					return i + 3
				}
				return i + 2
			}
		}
	} else {
		for i := l.position + 2; i < len(l.input); i++ {
			if l.input[i] == ']' {
				// Check if there's a quantifier after the closing bracket
				if i+1 < len(l.input) && isQuantifier(l.input[i+1]) {
					return i + 2
				}
				return i + 1
			}
		}
	}
	return -1
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
