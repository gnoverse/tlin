package fixerv2

import "strings"

type Replacer struct {
	patternNodes     []Node
	replacementNodes []Node
	baseIndent       string
}

func NewReplacer(pattern, replacement []Node) *Replacer {
	return &Replacer{
		patternNodes:     pattern,
		replacementNodes: replacement,
	}
}

func (r *Replacer) ReplaceAll(subject string) string {
	result := ""
	pos := 0

	for {
		found, matchStart, matchEnd, captures := findNextMatch(r.patternNodes, subject, pos)
		if !found {
			result += subject[pos:]
			break
		}

		result += subject[pos:matchStart]

		// Update base indent when match is found
		if idx := strings.LastIndex(subject[:matchStart], "\n"); idx != -1 {
			r.baseIndent = subject[idx+1 : matchStart]
		} else {
			r.baseIndent = ""
		}

		rawRepl := r.applyReplacement(captures)
		adjusted := r.adjustIndent(rawRepl)
		result += adjusted
		pos = matchEnd
	}
	return result
}

func (r *Replacer) applyReplacement(captures map[string]string) string {
	result := ""
	for _, node := range r.replacementNodes {
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

func (r *Replacer) adjustIndent(repl string) string {
	lines := strings.Split(repl, "\n")
	for i := 1; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], r.baseIndent) {
			lines[i] = r.baseIndent + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}
