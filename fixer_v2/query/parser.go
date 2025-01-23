package query

import (
	"fmt"
)

// hole name -> position (optional usage)
type holes map[string]int

// Parser is supposed to consume tokens produced by the lexer and build an AST.
type Parser struct {
	buffer  *buffer
	current int
	tokens  []Token
	holes   holes
}

func NewParser() *Parser {
	return &Parser{holes: make(holes)}
}

func (p *Parser) Parse(buf *buffer) ([]Node, error) {
	p.buffer = buf
	p.tokens = []Token{}

	err := p.collectTokens()
	if err != nil {
		return nil, fmt.Errorf("failed to collect tokens: %w", err)
	}

	rootNode := &PatternNode{}
	current := 0

	for current < len(p.tokens) {
		if p.tokens[current].Type == TokenEOF {
			break
		}

		node := p.parseTokenNode(current)
		if node != nil {
			rootNode.Children = append(rootNode.Children, node)
		}
		current++
	}

	return rootNode.Children, nil
}

func (p *Parser) collectTokens() error {
	for {
		token, err := p.nextToken()
		if err != nil {
			return err
		}

		p.tokens = append(p.tokens, token)

		if token.Type == TokenEOF {
			break
		}
	}
	return nil
}

func (p *Parser) nextToken() (Token, error) {
	if p.buffer.index >= p.buffer.length {
		return Token{Type: TokenEOF}, nil
	}

	class := p.buffer.getClass()

	switch class {
	case C_COLON:
		return p.scanMetaVariable()

	case C_LBRACE, C_RBRACE:
		return p.scanBrace()

	default:
		// treat everything else as text (or with WS) for now
		return p.scanText()
	}
}

func (p *Parser) scanMetaVariable() (Token, error) {
	startPos := p.buffer.index

	if p.buffer.data[startPos] != ':' {
		return Token{}, fmt.Errorf("expected ':' at position %d", startPos)
	}

	if startPos+1 >= p.buffer.length || p.buffer.data[startPos+1] != '[' {
		return Token{}, fmt.Errorf("expected '[' at position %d", startPos+1)
	}

	p.buffer.setMode(ModeHole)
	defer p.buffer.setMode(ModeText)

	cfg, err := p.buffer.parseMetaVariable()
	if err != nil {
		return Token{}, fmt.Errorf("failed to parse meta variable at position %d: %w", startPos, err)
	}

	return Token{
		Type:       TokenHole,
		Value:      p.buffer.tokenValue.String(),
		Position:   startPos,
		HoleConfig: cfg,
	}, nil
}

func (p *Parser) scanWhitespace() (Token, error) {
	text, err := p.buffer.parseText()
	if err != nil {
		return Token{}, err
	}

	return Token{
		Type:     TokenWhitespace,
		Value:    text,
		Position: p.buffer.tokenStart,
	}, nil
}

func (p *Parser) scanText() (Token, error) {
	text, err := p.buffer.parseText()
	if err != nil {
		return Token{}, err
	}

	if text == "" {
		return p.nextToken()
	}

	return Token{
		Type:     TokenText,
		Value:    text,
		Position: p.buffer.tokenStart,
	}, nil
}

func (p *Parser) scanBrace() (Token, error) {
	c := p.buffer.data[p.buffer.index]
	p.buffer.index++

	tt := TokenLBrace
	if c == '}' {
		tt = TokenRBrace
	}

	return Token{
		Type:     tt,
		Value:    string(c),
		Position: p.buffer.index - 1,
	}, nil
}

func (p *Parser) parseTokenNode(current int) Node {
	token := p.tokens[current]

	switch token.Type {
	case TokenText, TokenWhitespace:
		return &TextNode{
			Content: token.Value,
			pos:     token.Position,
		}
	case TokenHole:
		if token.HoleConfig != nil {
			p.holes[token.HoleConfig.Name] = token.Position
			return &HoleNode{
				Config: *token.HoleConfig,
				pos:    token.Position,
			}
		}

		holeName := extractHoleName(token.Value)
		p.holes[holeName] = token.Position
		return NewHoleNode(holeName, token.Position)

	case TokenLBrace:
		return p.parseBlockFromTokens(current)

	default:
		return nil
	}
}

func (p *Parser) parseBlockFromTokens(start int) Node {
	bn := &BlockNode{
		Content: make([]Node, 0),
		pos:     p.tokens[start].Position,
	}

	current := start + 1
	for current < len(p.tokens) {
		if p.tokens[current].Type == TokenRBrace {
			return bn
		}

		if node := p.parseTokenNode(current); node != nil {
			bn.Content = append(bn.Content, node)
		}
		current++
	}

	return bn
}
