package fixerv2

import (
	"fmt"
	"regexp"
	"strings"

	parser "github.com/gnolang/tlin/fixer_v2/query"
)

// Pattern represents a pattern-rewrite pair for code transformation
type Pattern struct {
	Match   string
	Rewrite string
}

var (
	whitespaceRegex = regexp.MustCompile(`\s+`)
	openBraceRegex  = regexp.MustCompile(`\s*{\s*`)
	closeBraceRegex = regexp.MustCompile(`\s*}\s*`)
)

// normalizePattern replaces consecutive whitespaces with a single space
// and standardizes the spacing around curly braces.
// Then it trims any leading or trailing whitespace.
// This helps unify the style of the pattern for regex generation.
//
// Note: this function is only used for testing
func normalizePattern(pattern string) string {
	pattern = whitespaceRegex.ReplaceAllString(pattern, " ")
	pattern = openBraceRegex.ReplaceAllString(pattern, " { ")
	pattern = closeBraceRegex.ReplaceAllString(pattern, " } ")
	return strings.TrimSpace(pattern)
}

// buildRegexFromAST builds a regex pattern from the parsed AST
func buildRegexFromAST(node parser.Node) Option[Result] {
	var sb strings.Builder
	captures := make(map[string]int)
	groupCount := 1

	var processNode func(parser.Node)
	processNode = func(n parser.Node) {
		switch v := n.(type) {
		case *parser.TextNode:
			// treat text nodes as literals and convert whitespace to \s+
			escaped := regexp.QuoteMeta(v.Content)
			processed := regexp.MustCompile(`\s+`).ReplaceAllString(escaped, `\s+`)
			sb.WriteString(processed)

		case *parser.HoleNode:
			// convert hole name to capture group name
			captures[v.Config.Name] = groupCount
			groupCount++
			sb.WriteString(`([^{}]+?)`)

		case *parser.BlockNode:
			// block nodes contain curly braces and handle internal nodes
			sb.WriteString(`\s*{\s*`)
			for _, child := range v.Content {
				processNode(child)
			}
			sb.WriteString(`\s*}\s*`)

		case *parser.PatternNode:
			// pattern nodes traverse all child nodes
			for _, child := range v.Children {
				processNode(child)
			}
		}
	}

	processNode(node)

	regex, err := regexp.Compile(sb.String())
	return createOption(Result{regex: regex, captures: captures}, err)
}

// patternToRegex converts the pattern string to a compiled *regexp.Regexp
// and returns a Result containing the regex and a map that correlates each
// placeholder name with its capture group index.
func patternToRegex(pattern string) Option[Result] {
	lexer := parser.NewLexer(pattern)
	tokens := lexer.Tokenize()

	parser := parser.NewParser(tokens)
	ast := parser.Parse()

	return buildRegexFromAST(ast)
}

// rewrite replaces placeholders in the rewrite pattern with the captured values in 'env'.
//
// For each placeholder name, we look for :[[name]] or :[name] in rewritePattern
// and substitute with the corresponding 'env[name]' value.
func rewrite(rewritePattern string, env map[string]string) string {
	lexer := parser.NewLexer(rewritePattern)
	tokens := lexer.Tokenize()

	prsr := parser.NewParser(tokens)
	ast := prsr.Parse()

	var result strings.Builder

	var processNode func(parser.Node)
	processNode = func(n parser.Node) {
		switch v := n.(type) {
		case *parser.TextNode:
			result.WriteString(v.Content)

		case *parser.HoleNode:
			// replace hole name with the corresponding value in 'env'
			if value, ok := env[v.Config.Name]; ok {
				result.WriteString(value)
			} else {
				// if value is not found, keep the original hole expression
				result.WriteString(fmt.Sprintf(":[%s]", v.Config.Name))
			}

		case *parser.BlockNode:
			result.WriteString("{")
			for _, child := range v.Content {
				processNode(child)
			}
			result.WriteString("}")

		case *parser.PatternNode:
			for _, child := range v.Children {
				processNode(child)
			}
		}
	}

	processNode(ast)
	return result.String()
}
