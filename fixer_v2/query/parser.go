package query

// Parser is supposed to consume tokens produced by the lexer and build an AST.
type Parser struct {
	tokens  []Token
	current int
	holes   map[string]int // hole name -> position (optional usage)
}

// NewParser creates a new Parser instance
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens:  tokens,
		current: 0,
		holes:   make(map[string]int),
	}
}

// Parse processes all tokens and builds an AST
func (p *Parser) Parse() Node {
	rootNode := &PatternNode{pos: 0}

	for p.current < len(p.tokens) {
		if p.tokens[p.current].Type == TokenEOF {
			break
		}

		node := p.parseNode()
		if node != nil {
			rootNode.Children = append(rootNode.Children, node)
		}
	}

	return rootNode
}

// parseNode parses a single node based on the current token
func (p *Parser) parseNode() Node {
	token := p.tokens[p.current]

	switch token.Type {
	case TokenText, TokenWhitespace:
		p.current++
		return &TextNode{
			Content: token.Value,
			pos:     token.Position,
		}
	case TokenHole:
		p.current++
		holeName := extractHoleName(token.Value)
		// update hole expr's position
		p.holes[holeName] = token.Position
		return &HoleNode{
			Name: holeName,
			pos:  token.Position,
		}
	case TokenLBrace:
		return p.parseBlock()
	default:
		p.current++
		return nil
	}
}

// parseBlock parses a block enclosed by '{' and '}'
func (p *Parser) parseBlock() Node {
	openPos := p.tokens[p.current].Position
	p.current++

	blockNode := &BlockNode{
		Content: make([]Node, 0),
		pos:     openPos,
	}

	// parse nodes until we find the matching '}'
	for p.current < len(p.tokens) {
		if p.tokens[p.current].Type == TokenRBrace {
			p.current++
			return blockNode
		}

		node := p.parseNode()
		if node != nil {
			blockNode.Content = append(blockNode.Content, node)
		}
	}

	// if we get here, we're missing a closing brace
	// TODO: error handling
	return blockNode
}

// peek peeks at the next token
// TODO: commented out for now
// func (p *Parser) peek() Token {
// 	if p.current+1 >= len(p.tokens) {
// 		return Token{Type: TokenEOF, Value: "", Position: -1}
// 	}
// 	return p.tokens[p.current+1]
// }
