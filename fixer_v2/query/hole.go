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

var quantifiers = map[byte]bool{
	'*': true, '+': true, '?': true,
}

// isQuantifier checks if a character is a valid quantifier
func isQuantifier(c byte) bool {
	return quantifiers[c]
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
