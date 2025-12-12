package syntax

import "fmt"

// parseNames parses tables declare tag from an anonymous field
func parseNames(tag string) ([]string, error) {
	var names []string
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
				return nil, fmt.Errorf("empty table name at position %d", i)
			}
			if isDigit(r) {
				return nil, fmt.Errorf("invalid table name starting with %q at position %d", r, i)
			}
			if !isAllowedNameChar(r) {
				return nil, fmt.Errorf("invalid character %q in table name at position %d", r, i)
			}
			start = i
			state = 1
		case 1: // in token
			if r == ',' {
				names = append(names, tag[start:i])
				state = 0
			} else if r == ' ' || r == '\t' {
				names = append(names, tag[start:i])
				state = 2
			} else if !isAllowedNameChar(r) {
				return nil, fmt.Errorf("invalid character %q in table name at position %d", r, i)
			}
		case 2: // after token, waiting for comma
			if r == ',' {
				state = 0
			} else if r == ' ' || r == '\t' {
				continue
			} else {
				return nil, fmt.Errorf("invalid character %q after table name at position %d, expecting a comma", r, i)
			}
		}
	}
	if state == 0 { // ended while waiting for token
		return nil, fmt.Errorf("empty table name at the end of tag")
	}
	if state == 1 { // ended while in token
		names = append(names, tag[start:])
	}
	return names, nil
}
