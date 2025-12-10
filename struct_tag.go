package sqlb

import (
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// parseTag parses the sqlb tag from a struct field.
// tag formats:
// 1. <table>.<column>
// 2. <expression>;[table1, table2...]
// e.g. "u.id", "COALESCE(?.id,?.user_id,0);u,j"
func parseTag(tag string, declaredTables []any) (column sqlf.Builder, err error) {
	table, col, ok := parseSimpleTag(tag)
	if ok {
		t := NewTable("", table)
		return t.Column(col), nil
	}
	var tables []any
	posSep := strings.IndexRune(tag, ';')
	expr := tag
	if posSep >= 0 {
		expr = tag[:posSep]
		tableNames, err := parseDeclareTag(tag, posSep+1)
		if err != nil {
			return nil, err
		}
		tables = util.Map(tableNames, func(t string) any {
			return NewTable("", strings.TrimSpace(t))
		})
	}
	useDeclared := false
	if len(tables) == 0 {
		useDeclared = true
		tables = declaredTables
	}
	f := sqlf.F(expr, tables...)
	if useDeclared {
		f.NoUsageCheck()
	}
	column = f
	// try build column to catch errors early for better error messages
	ctx := sqlf.NewContext(sqlf.BindStyleDollar)
	_, err = column.Build(ctx)
	if err != nil {
		return nil, err
	}
	return column, nil
}

// parseSimpleTag parses a simple sqlb tag in the format of "<table>.<column>".
func parseSimpleTag(tag string) (tableName, columnName string, ok bool) {
	var indexDot int
	for i, ch := range tag {
		if i == 0 && (isDigit(ch) || ch == '.') {
			// starting with a digit or dot
			return "", "", false
		}
		if ch == '.' {
			if indexDot > 0 {
				// multiple dot
				return "", "", false
			}
			indexDot = i
			tableName = tag[:i]
			continue
		}
		if !isAllowedNameChar(ch) {
			return "", "", false
		}
	}
	if indexDot == 0 || indexDot == len(tag)-1 {
		// no dot or dot at the end
		return "", "", false
	}
	columnName = tag[indexDot+1:]
	return tableName, columnName, true
}

// parseDeclareTag parses tables declare tag from an anonymous field
func parseDeclareTag(tag string, offset int) (tables []string, err error) {
	if offset > 0 {
		if offset >= len(tag) {
			return nil, nil
		}
		tag = tag[offset:]
	}
	var start int
	// 0: waiting for token
	// 1: in token
	// 2: after token, waiting for comma
	state := 0
	for i, r := range tag {
		switch state {
		case 0: // waiting for token
			if r == ' ' || r == '\t' {
				continue
			}
			if r == ',' {
				return nil, fmt.Errorf("empty table name at position %d", i+offset)
			}
			if isDigit(r) {
				return nil, fmt.Errorf("invalid table name starting with %q at position %d", r, i+offset)
			}
			if !isAllowedNameChar(r) {
				return nil, fmt.Errorf("invalid character %q in table name at position %d", r, i+offset)
			}
			start = i
			state = 1
		case 1: // in token
			if r == ',' {
				tables = append(tables, tag[start:i])
				state = 0
			} else if r == ' ' || r == '\t' {
				tables = append(tables, tag[start:i])
				state = 2
			} else if !isAllowedNameChar(r) {
				return nil, fmt.Errorf("invalid character %q in table name at position %d", r, i+offset)
			}
		case 2: // after token, waiting for comma
			if r == ',' {
				state = 0
			} else if r == ' ' || r == '\t' {
				continue
			} else {
				return nil, fmt.Errorf("invalid character %q after table name at position %d, expecting a comma", r, i+offset)
			}
		}
	}
	if state == 0 { // ended while waiting for token
		return nil, fmt.Errorf("empty table name at the end of tag")
	}
	if state == 1 { // ended while in token
		tables = append(tables, tag[start:])
	}
	return tables, nil
}

func isAllowedNameChar(ch rune) bool {
	return ch >= 'a' && ch <= 'z' ||
		ch >= 'A' && ch <= 'Z' ||
		ch >= '0' && ch <= '9' ||
		ch == '_' || ch == '@' || ch == '#'
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
