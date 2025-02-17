package fixerv2

import "maps"

// TODO: Refactor this
// TODO: Needs to be created whitespace-characters-free in the pattern and subject.

// Match checks if the entire subject matches the pattern
func Match(nodes []Node, subject string) (bool, map[string]string) {
	ok, end, captures := matcher(nodes, 0, subject, 0, map[string]string{})
	if ok && end == len(subject) {
		return true, captures
	}
	return false, nil
}

// matcher attempts to match pattern nodes with subject starting from sIdx using recursive backtracking.
func matcher(nodes []Node, pIdx int, subject string, sIdx int, captures map[string]string) (bool, int, map[string]string) {
	if pIdx == len(nodes) {
		return true, sIdx, captures
	}

	currentNode := nodes[pIdx]
	switch node := currentNode.(type) {
	case LiteralNode:
		return matchLiteral(nodes, pIdx, node, subject, sIdx, captures)
	case MetaVariableNode:
		if node.Ellipsis {
			return matchEllipsis(nodes, pIdx, subject, sIdx, captures)
		}
		return matchMetaVariable(nodes, pIdx, node, subject, sIdx, captures)
	default:
		return false, 0, nil
	}
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

// matchMetaVariable handles matching of meta variable nodes
func matchMetaVariable(nodes []Node, pIdx int, node MetaVariableNode, subject string, sIdx int, captures map[string]string) (bool, int, map[string]string) {
	if pIdx == len(nodes)-1 {
		return matchLastMetaVariable(node, subject, sIdx, captures)
	}
	return matchMiddleMetaVariable(nodes, pIdx, node, subject, sIdx, captures)
}

// matchLastMetaVariable handles meta variable that is the last node in the pattern
func matchLastMetaVariable(node MetaVariableNode, subject string, sIdx int, captures map[string]string) (bool, int, map[string]string) {
	for k := sIdx + 1; k <= len(subject); k++ {
		if ok, end, caps := tryMatchLastMetaVariable(node, subject, sIdx, k, captures); ok {
			return true, end, caps
		}
	}
	return false, 0, nil
}

// tryMatchLastMetaVariable attempts to match a candidate for the last meta variable
func tryMatchLastMetaVariable(node MetaVariableNode, subject string, sIdx, k int, captures map[string]string) (bool, int, map[string]string) {
	candidate := subject[sIdx:k]
	newCaptures := copyCaptures(captures)
	newCaptures[node.Name] = candidate

	if k < len(subject) {
		nextChar := subject[k]
		if isNumeric(candidate) {
			if !isDigit(nextChar) {
				return true, k, newCaptures
			}
		} else if isHardDelimiter(nextChar) {
			return true, k, newCaptures
		}
	} else {
		return true, k, newCaptures
	}

	return false, 0, nil
}

// matchMiddleMetaVariable handles meta variable that is not the last node
func matchMiddleMetaVariable(nodes []Node, pIdx int, node MetaVariableNode, subject string, sIdx int, captures map[string]string) (bool, int, map[string]string) {
	delimiter := findNextDelimiter(nodes, pIdx+1)
	if delimiter != "" {
		return matchWithDelimiter(nodes, pIdx, node, subject, sIdx, captures, delimiter)
	}
	return matchWithoutDelimiter(nodes, pIdx, node, subject, sIdx, captures)
}

// matchLiteral handles matching of literal nodes
func matchLiteral(nodes []Node, pIdx int, node LiteralNode, subject string, sIdx int, captures map[string]string) (bool, int, map[string]string) {
	lit := node.Value
	if sIdx+len(lit) > len(subject) || subject[sIdx:sIdx+len(lit)] != lit {
		return false, 0, nil
	}
	return matcher(nodes, pIdx+1, subject, sIdx+len(lit), captures)
}

// findNextDelimiter finds the next non-empty literal value in the pattern
func findNextDelimiter(nodes []Node, startIdx int) string {
	for j := startIdx; j < len(nodes); j++ {
		if litNode, ok := nodes[j].(LiteralNode); ok && litNode.Value != "" {
			return litNode.Value
		}
	}
	return ""
}

// matchWithDelimiter handles meta variable matching when a delimiter is found
func matchWithDelimiter(nodes []Node, pIdx int, node MetaVariableNode, subject string, sIdx int, captures map[string]string, delimiter string) (bool, int, map[string]string) {
	for k := sIdx + 1; k <= len(subject)-len(delimiter); k++ {
		if subject[k:k+len(delimiter)] == delimiter {
			if ok, end, caps := tryMatchWithDelimiter(nodes, pIdx, node, subject, sIdx, k, captures); ok {
				return true, end, caps
			}
		}
	}
	return false, 0, nil
}

// tryMatchWithDelimiter attempts to match a candidate when a delimiter is present
func tryMatchWithDelimiter(nodes []Node, pIdx int, node MetaVariableNode, subject string, sIdx, k int, captures map[string]string) (bool, int, map[string]string) {
	newCaptures := copyCaptures(captures)
	newCaptures[node.Name] = subject[sIdx:k]
	return matcher(nodes, pIdx+1, subject, k, newCaptures)
}

// matchWithoutDelimiter handles meta variable matching when no delimiter is found
func matchWithoutDelimiter(nodes []Node, pIdx int, node MetaVariableNode, subject string, sIdx int, captures map[string]string) (bool, int, map[string]string) {
	for k := sIdx + 1; k <= len(subject); k++ {
		if ok, end, caps := tryMatchWithoutDelimiter(nodes, pIdx, node, subject, sIdx, k, captures); ok {
			return true, end, caps
		}
	}
	return false, 0, nil
}

// tryMatchWithoutDelimiter attempts to match a candidate when no delimiter is present
func tryMatchWithoutDelimiter(nodes []Node, pIdx int, node MetaVariableNode, subject string, sIdx, k int, captures map[string]string) (bool, int, map[string]string) {
	newCaptures := copyCaptures(captures)
	newCaptures[node.Name] = subject[sIdx:k]
	return matcher(nodes, pIdx+1, subject, k, newCaptures)
}

var delimeterSet = map[byte]bool{
	'\n': true, '}': true, ';': true,
}

func isHardDelimiter(c byte) bool {
	return delimeterSet[c]
}

// matchEllipsis handles MetaVariableNode with when node is ellipsis.
// We try to capture as much as needed until we can match the next node.
func matchEllipsis(nodes []Node, pIdx int, subject string, sIdx int, captures map[string]string) (bool, int, map[string]string) {
	// If this is the last node, greedily capture until the end.
	if pIdx == len(nodes)-1 {
		newCaps := copyCaptures(captures)
		newCaps[nodes[pIdx].(MetaVariableNode).Name] = subject[sIdx:]
		return true, len(subject), newCaps
	}

	// Otherwise, we need to find a chunk that allows the next node to match.
	// We'll attempt from sIdx+0 up to len(subject).
	metaName := nodes[pIdx].(MetaVariableNode).Name

	for cut := sIdx; cut <= len(subject); cut++ {
		// Try capturing subject[sIdx:cut] as the ellipsis body
		newCaps := copyCaptures(captures)
		newCaps[metaName] = subject[sIdx:cut]

		// Then try to match the next node from `cut`.
		ok, end, res := matcher(nodes, pIdx+1, subject, cut, newCaps)
		if ok {
			return true, end, res
		}
	}

	// No valid match found
	return false, 0, nil
}

// isNumeric returns true if s is non-empty and every character is a digit.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

// copyCaptures creates a shallow copy of the captures map
func copyCaptures(captures map[string]string) map[string]string {
	if len(captures) == 0 {
		return make(map[string]string)
	}
	return maps.Clone(captures)
}
