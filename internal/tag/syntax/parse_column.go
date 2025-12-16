package syntax

// NOT USED ANYMORE
func parseColumn(value string) (expr, table string) {
	table, column, ok := parseSimpleColumn(value)
	if ok {
		return "?." + column, table
	}
	return value, ""
}

func parseSimpleColumn(value string) (tableName, columnName string, ok bool) {
	var indexDot int
	for i, ch := range value {
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
			tableName = value[:i]
			continue
		}
		if !isAllowedNameChar(ch) {
			return "", "", false
		}
	}
	if indexDot == 0 || indexDot == len(value)-1 {
		// no dot or dot at the end
		return "", "", false
	}
	columnName = value[indexDot+1:]
	return tableName, columnName, true
}

func isAllowedName(name string) bool {
	if name == "" {
		return false
	}
	for _, ch := range name {
		if !isAllowedNameChar(ch) {
			return false
		}
	}
	return true
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
