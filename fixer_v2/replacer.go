package fixerv2

// ReplaceAll replaces all occurrences in the subject that match the pattern with the replacement template
func ReplaceAll(patternNodes []Node, replacementNodes []Node, subject string) string {
	result := ""
	pos := 0
	for {
		found, matchStart, matchEnd, captures := findNextMatch(patternNodes, subject, pos)
		if !found {
			result += subject[pos:]
			break
		}
		result += subject[pos:matchStart]
		result += applyReplacement(replacementNodes, captures)
		pos = matchEnd
	}
	return result
}

// applyReplacement generates a replacement string using the replacement template AST and capture map
func applyReplacement(replacementNodes []Node, captures map[string]string) string {
	result := ""
	for _, node := range replacementNodes {
		switch n := node.(type) {
		case LiteralNode:
			result += n.Value
		case MetaVariableNode:
			if val, ok := captures[n.Name]; ok {
				result += val
			}
		}
	}
	return result
}
