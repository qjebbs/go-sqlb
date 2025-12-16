package syntax

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Info represents parsed tag information.
type Info struct {
	Select        string   // Select is parsed from "sel" key.
	Column        string   // Column is parsed from "col" key.
	Tables        []string // Tables is parsed from "tables" key.
	DefaultTables []string // DefaultTables is parsed from "default_tables" key.
	On            []string // On is parsed from "on" key.
	Dive          bool     // Dive indicates whether "dive" key is present.
}

// Parse parses the input and returns the list of expressions.
func Parse(input string) (*Info, error) {
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
	p.NextToken()
	if p.token.typ == _Semicolon || p.token.typ == _EOF {
		switch key {
		case "dive":
			p.c.Dive = true
			return nil
		}
	}
	if p.token.typ != _Colon {
		return p.syntaxError(fmt.Sprintf("expected %s, see %s", _Colon, p.token.typ))
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
		value := strings.TrimSpace(p.token.lit)
		firstRune, _ := utf8.DecodeRuneInString(p.token.lit)
		// lastRune, _ := utf8.DecodeLastRuneInString(p.token.lit)
		if firstRune != '"' && firstRune != '`' && firstRune != '\'' && firstRune != '[' {
			// check name validity only for unquoted names
			if !isAllowedName(value) {
				return p.syntaxError(fmt.Sprintf("invalid column name, at %d: %q", p.token.start, value))
			}
		}
		p.c.Column = value
	case "sel":
		// Semantically: "col" emphasizes a column for write operations (e.g. INSERT/UPDATE),
		// while "sel" emphasizes an expression for SELECT queries. They are equivalent
		// for parsing and SELECT usage.
		//
		// If INSERT/UPDATE support is added later, both "col" and "sel" could be used in SELECT,
		// but only "col" would be valid for INSERT/UPDATE. Therefore, if a "col" tag is defined
		// it can substitute a "sel" tag for SELECT semantics.
		if p.c.Select != "" {
			return p.syntaxError(fmt.Sprintf("redundant select declaration, at %d: %q", p.token.start, p.token.lit))
		}
		p.c.Select = p.token.lit
	case "tables", "default_tables":
		names, err := parseNames(p.token.lit)
		if err != nil {
			return err
		}
		if len(p.c.Tables) > 0 {
			return p.syntaxError(fmt.Sprintf("redundant tables declaration, at %d: %q", p.token.start, p.token.lit))
		}
		if key == "tables" {
			p.c.Tables = names
		} else {
			p.c.DefaultTables = names
		}
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
