package syntax

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/qjebbs/go-sqlb/internal/util"
)

// Info represents parsed tag information.
type Info struct {
	Select string   // Select is parsed from "sel" key.
	Column string   // Column is parsed from "col" key.
	Table  string   // Table is parsed from "table" key.
	From   []string // From is parsed from "from" key.
	On     []string // On is parsed from "on" key.
	Dive   bool     // Dive indicates whether "dive" key is present.

	PK              bool    // PK indicates whether "pk" key is present.
	ReadOnly        bool    // ReadOnly indicates whether "readonly" key is present.
	InsertOmitEmpty bool    // InsertOmitEmpty indicates whether "insert_omitempty" key is present.
	Returning       bool    // Returning indicates whether "returning" key is present.
	ConflictOn      bool    // ConflictOn indicates whether "conflict_on" key is present.
	ConflictSet     *string // ConflictSet is parsed from "conflict_set" key.

	Unique bool // Unique indicates whether "unique" key is present.
	Match  bool // Match indicates whether "match" key is present.
}

// Parse parses the input and returns the list of expressions.
func Parse(input string) (*Info, error) {
	p := newParser(input)
	if err := p.Parse(); err != nil {
		return nil, err
	}
	return p.c, nil
}

type parser struct {
	*scanner
	c *Info

	state         parseFn
	presentedKeys map[string]bool
}

func newParser(input string) *parser {
	return &parser{
		scanner:       newScanner(input),
		c:             &Info{},
		state:         parse,
		presentedKeys: make(map[string]bool),
	}
}

type parseFn func(*parser) (parseFn, error)

func (p *parser) want(t ...TokenType) error {
	if !p.got(t...) {
		return p.syntaxError(
			fmt.Sprintf("unexpected %s, want %s", p.token.typ, strings.Join(util.Map(t, func(v TokenType) string { return string(v) }), ",")),
		)
	}
	return nil
}

func (p *parser) got(tok ...TokenType) bool {
	p.NextToken()
	for _, t := range tok {
		if p.token.typ == t {
			return true
		}
	}
	return false
}

func (p *parser) syntaxError(msg string) error {
	return fmt.Errorf("%d: syntax error: %s", p.token.start, msg)
}

func (p *parser) Parse() error {
	for p.state != nil {
		s, err := p.state(p)
		if err != nil {
			return err
		}
		p.state = s
	}
	return nil
}

func parse(p *parser) (parseFn, error) {
	p.NextToken()
	switch p.token.typ {
	case _EOF:
		return nil, nil
	case _Error:
		return nil, p.syntaxError("invalid syntax")
	case _Key:
		return parseKeyValue, nil
	default:
		return nil, p.syntaxError("unexpected token " + string(p.token.typ))
	}
}

func parseKeyValue(p *parser) (parseFn, error) {
	key := p.token.lit
	if p.presentedKeys[key] {
		return nil, p.syntaxError(fmt.Sprintf("duplicated key %q at %d: %q", key, p.token.start, p.token.lit))
	}
	switch key {
	case "dive":
		return parseBoolAndSet(p, func(v bool) {
			p.c.Dive = v
		})
	case "col":
		return parseStringAndSet(p, func(v string) error {
			firstRune, _ := utf8.DecodeRuneInString(p.token.lit)
			// lastRune, _ := utf8.DecodeLastRuneInString(p.token.lit)
			if firstRune != '"' && firstRune != '`' && firstRune != '\'' && firstRune != '[' {
				// check name validity only for unquoted names
				if !isAllowedName(v) {
					return p.syntaxError(fmt.Sprintf("invalid column name, at %d: %q", p.token.start, v))
				}
			}
			p.c.Column = v
			return nil
		})

	case "sel":
		// Semantically: "col" emphasizes a column for write operations (e.g. INSERT/UPDATE),
		// while "sel" emphasizes an expression for SELECT queries. They are equivalent
		// for parsing and SELECT usage.
		//
		// If INSERT/UPDATE support is added later, both "col" and "sel" could be used in SELECT,
		// but only "col" would be valid for INSERT/UPDATE. Therefore, if a "col" tag is defined
		// it can substitute a "sel" tag for SELECT semantics.
		return parseStringAndSet(p, func(v string) error {
			p.c.Select = p.token.lit
			return nil
		})
	case "table":
		return parseStringAndSet(p, func(v string) error {
			p.c.Table = v
			return nil
		})
	case "from":
		return parseStringAndSet(p, func(v string) error {
			names, err := parseNames(v)
			if err != nil {
				return err
			}
			p.c.From = names
			return nil
		})
	case "on":
		return parseStringAndSet(p, func(v string) error {
			names, err := parseNames(v)
			if err != nil {
				return err
			}
			p.c.On = names
			return nil
		})
	case "pk":
		return parseBoolAndSet(p, func(v bool) {
			p.c.PK = v
		})
	case "unique":
		return parseBoolAndSet(p, func(v bool) {
			p.c.Unique = v
		})
	case "match":
		return parseBoolAndSet(p, func(v bool) {
			p.c.Match = v
		})
	case "readonly":
		return parseBoolAndSet(p, func(v bool) {
			p.c.ReadOnly = v
		})
	case "insert_omitempty":
		return parseBoolAndSet(p, func(v bool) {
			p.c.InsertOmitEmpty = v
		})
	case "returning":
		return parseBoolAndSet(p, func(v bool) {
			p.c.Returning = v
		})
	case "conflict_on":
		return parseBoolAndSet(p, func(v bool) {
			p.c.ConflictOn = v
		})
	case "conflict_set":
		return parseStringPtrAndSet(p, func(v *string) error {
			p.c.ConflictSet = v
			return nil
		})
	default:
		return nil, p.syntaxError("unknown key: " + key)
	}
}

// parseBoolAndSet parses a boolean value and sets it using the provided setter function.
// Accepted syntax:
//   - key; (indicates true)
func parseBoolAndSet(p *parser, set func(v bool)) (parseFn, error) {
	// bool key does not have value, its presence indicates true
	if err := p.want(_Semicolon, _EOF); err != nil {
		return nil, err
	}
	set(true)
	return parse, nil
}

// parseStringAndSet parses a string value and sets it using the provided setter function.
// Accepted syntax:
//   - key:value;
func parseStringAndSet(p *parser, set func(v string) error) (parseFn, error) {
	if err := p.want(_Colon); err != nil {
		return nil, err
	}
	if err := p.want(_Value); err != nil {
		return nil, err
	}
	if err := set(p.token.lit); err != nil {
		return nil, err
	}
	if err := p.want(_Semicolon, _EOF); err != nil {
		return nil, err
	}
	return parse, nil
}

// parseStringPtrAndSet parses a string value or nil and sets it using the provided setter function.
// Accepted syntax:
//   - key:value;
//   - key; (sets nil)
func parseStringPtrAndSet(p *parser, set func(v *string) error) (parseFn, error) {
	if err := p.want(_Colon, _Semicolon, _EOF); err != nil {
		return nil, err
	}
	if p.token.typ != _Colon {
		// presence, set empty
		empty := ""
		if err := set(&empty); err != nil {
			return nil, err
		}
		return parse, nil
	}
	if err := p.want(_Value); err != nil {
		return nil, err
	}
	value := p.token.lit
	if err := set(&value); err != nil {
		return nil, err
	}
	if err := p.want(_Semicolon, _EOF); err != nil {
		return nil, err
	}
	return parse, nil
}
