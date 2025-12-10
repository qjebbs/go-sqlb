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
func parseTag(tag string) (sqlf.Builder, error) {
	table, col, ok := parseSimpleTag(tag)
	if ok {
		return NewTable("", table).Column(col), nil
	}
	seg := strings.SplitN(tag, ";", 2)
	var tables []any
	if len(seg) == 2 {
		tableNames := strings.Split(seg[1], ",")
		tables = util.Map(tableNames, func(t string) any {
			return NewTable("", strings.TrimSpace(t))
		})
	}
	column := sqlf.F(seg[0], tables...)
	// try build column to catch errors early for better error messages
	ctx := sqlf.NewContext(sqlf.BindStyleDollar)
	_, err := column.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid sqlb tag %q: %w", tag, err)
	}
	return column, nil
}

// parseSimpleTag parses a simple sqlb tag in the format of "<table>.<column>".
func parseSimpleTag(tag string) (tableName, columnName string, ok bool) {
	var indexDot int
	for i, ch := range tag {
		if i == 0 && (ch >= '0' && ch <= '9' || ch == '.') {
			// starting with a digit
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
		if !(ch >= 'a' && ch <= 'z' ||
			ch >= 'A' && ch <= 'Z' ||
			ch >= '0' && ch <= '9' ||
			ch == '_' || ch == '@' || ch == '#') {
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
