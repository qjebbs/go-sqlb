package syntax

import (
	"fmt"
)

// Info represents parsed tag information.
type Info struct {
	Column string   // Column is parsed from "col" key.
	Tables []string // Tables is parsed from "tables" key.
	On     []string // On is parsed from "on" key.
}

// Parse parses the input and returns the list of expressions.
func Parse(input string) (*Info, error) {
	if t, col, ok := parseSimpleColumn(input); ok {
		return &Info{
			Column: "?." + col,
			Tables: []string{t},
		}, nil
	}
	p := &parser{
		scanner: newScanner(input),
	}
	if err := p.Parse(); err != nil {
		return nil, err
	}
	return p.c, nil
}

type parser struct {
	*scanner
	c *Info
}

func (p *parser) want(t TokenType) error {
	if !p.got(t) {
		return p.syntaxError(
			fmt.Sprintf("unexpected %s, want %s", p.token.typ, t),
		)
	}
	return nil
}

func (p *parser) got(tok TokenType) bool {
	p.NextToken()
	if p.token.typ == tok {
		return true
	}
	return false
}

func (p *parser) syntaxError(msg string) error {
	return fmt.Errorf("%d: syntax error: %s", p.token.start, msg)
}

func (p *parser) Parse() error {
	p.c = &Info{}
L:
	for p.NextToken() {
		switch p.token.typ {
		case _EOF:
			break L
		case _Error:
			return p.syntaxError("invalid syntax")
		case _Key:
			err := p.parseKeyValue()
			if err != nil {
				return err
			}
		default:
			return p.syntaxError("unexpected token " + string(p.token.typ))
		}
	}
	return nil
}

func (p *parser) parseKeyValue() error {
	key := p.token.lit
	if err := p.want(_Colon); err != nil {
		return err
	}
	p.NextToken()
	if p.token.typ == _Semicolon || p.token.typ == _EOF {
		// Empty value
		return nil
	}
	switch key {
	case "col":
		if p.c.Column != "" {
			return p.syntaxError(fmt.Sprintf("redundant column declaration, at %d: %q", p.token.start, p.token.lit))
		}
		expr, table := parseColumn(p.token.lit)
		p.c.Column = expr
		if table != "" {
			if len(p.c.Tables) > 0 {
				return p.syntaxError(fmt.Sprintf("redundant table declaration, at %d: %q", p.token.start, p.token.lit))
			}
			p.c.Tables = []string{table}
		}
	case "tables":
		names, err := parseNames(p.token.lit)
		if err != nil {
			return err
		}
		if len(p.c.Tables) > 0 {
			return p.syntaxError(fmt.Sprintf("redundant tables declaration, at %d: %q", p.token.start, p.token.lit))
		}
		p.c.Tables = names
	case "on":
		names, err := parseNames(p.token.lit)
		if err != nil {
			return err
		}
		p.c.On = names
	default:
		return p.syntaxError("unknown key: " + key)
	}
	p.NextToken()
	if p.token.typ != _EOF && p.token.typ != _Semicolon {
		return p.syntaxError(fmt.Sprintf("expected %s, see %s", _Semicolon, p.token.typ))
	}
	return nil
}
