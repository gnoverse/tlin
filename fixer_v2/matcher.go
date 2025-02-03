package fixerv2

// Match checks if the entire subject matches the pattern
func Match(nodes []Node, subject string) (bool, map[string]string) {
	ok, end, captures := matcher(nodes, 0, subject, 0, map[string]string{})
	if ok && end == len(subject) {
		return true, captures
	}
	return false, nil
}

// isNumeric returns true if s is non-empty and every character is a digit.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// matcher attempts to match pattern nodes (slice) with subject starting from sIdx using recursive backtracking.
// On success, returns (true, matching end index, capture map)
func matcher(nodes []Node, pIdx int, subject string, sIdx int, captures map[string]string) (bool, int, map[string]string) {
	if pIdx == len(nodes) {
		return true, sIdx, captures
	}

	switch node := nodes[pIdx].(type) {
	case LiteralNode:
		lit := node.Value
		if sIdx+len(lit) > len(subject) || subject[sIdx:sIdx+len(lit)] != lit {
			return false, 0, nil
		}
		return matcher(nodes, pIdx+1, subject, sIdx+len(lit), captures)

	case MetaVariableNode:
		// If this is the last metavariable
		if pIdx == len(nodes)-1 {
			// Try candidates with at least 1 character. For numbers, use minimum matching (i.e., decide candidate when next character is not a number).
			// For non-numbers, consider candidates up to the end of subject (or hard delimiters: newline, '}', ';')
			for k := sIdx + 1; k <= len(subject); k++ {
				candidate := subject[sIdx:k]
				newCaptures := copyCaptures(captures)
				newCaptures[node.Name] = candidate
				if k < len(subject) {
					nextChar := subject[k]
					if isNumeric(candidate) {
						// If candidate is a number, consider it a valid boundary when next character is not a number
						if !('0' <= nextChar && nextChar <= '9') {
							return true, k, newCaptures
						}
					} else {
						// If not a number, consider newline, '}', ';' as boundaries
						if nextChar == '\n' || nextChar == '}' || nextChar == ';' {
							return true, k, newCaptures
						}
					}
				} else {
					// Accept candidate when reaching the end of subject
					return true, k, newCaptures
				}
			}
			return false, 0, nil
		}

		// If not the last node - search for candidates based on following literal (delimiter)
		delimiter := ""
		found := false
		for j := pIdx + 1; j < len(nodes); j++ {
			if litNode, ok := nodes[j].(LiteralNode); ok && litNode.Value != "" {
				delimiter = litNode.Value
				found = true
				break
			}
		}
		if found {
			for k := sIdx + 1; k <= len(subject); k++ {
				if k+len(delimiter) > len(subject) {
					continue
				}
				if subject[k:k+len(delimiter)] == delimiter {
					candidate := subject[sIdx:k]
					newCaptures := copyCaptures(captures)
					newCaptures[node.Name] = candidate
					if ok, end, res := matcher(nodes, pIdx+1, subject, k, newCaptures); ok {
						return true, end, res
					}
				}
			}
			return false, 0, nil
		} else {
			// If no delimiter found: try all candidates with at least 1 character
			for k := sIdx + 1; k <= len(subject); k++ {
				candidate := subject[sIdx:k]
				newCaptures := copyCaptures(captures)
				newCaptures[node.Name] = candidate
				if ok, end, res := matcher(nodes, pIdx+1, subject, k, newCaptures); ok {
					return true, end, res
				}
			}
			return false, 0, nil
		}

	default:
		return false, 0, nil
	}
}

func copyCaptures(captures map[string]string) map[string]string {
	newMap := make(map[string]string)
	for k, v := range captures {
		newMap[k] = v
	}
	return newMap
}

// findNextMatch finds the leftmost match (partial match) in the subject after the start index
func findNextMatch(patternNodes []Node, subject string, start int) (bool, int, int, map[string]string) {
	for i := start; i < len(subject); i++ {
		// If pattern starts with `Meta`, assume matching start character must be a number (or alphabet)
		if len(patternNodes) > 0 {
			if _, ok := patternNodes[0].(MetaVariableNode); ok {
				if !isIdentifierChar(subject[i]) {
					continue
				}
			}
		}
		if ok, end, captures := matcher(patternNodes, 0, subject, i, map[string]string{}); ok {
			return true, i, end, captures
		}
	}
	return false, 0, 0, nil
}
