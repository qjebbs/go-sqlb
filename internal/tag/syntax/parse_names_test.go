package syntax

import "testing"

func TestParseDeclareTag(t *testing.T) {
	testCases := []struct {
		tag            string
		expectedTables []string
		expectError    bool
	}{
		{"u", []string{"u"}, false},
		{" u ", []string{"u"}, false},
		{"u, p", []string{"u", "p"}, false},
		{"  u    ,  p  ,  o  ", []string{"u", "p", "o"}, false},
		{",p", nil, true},        // empty table name
		{"u,,o", nil, true},      // empty table name
		{"u,", nil, true},        // empty table name
		{"1table", nil, true},    // starts with digit
		{"table$", nil, true},    // invalid character
		{"table, p$", nil, true}, // invalid character
		{"a b", nil, true},       // invalid character
	}

	for _, tc := range testCases {
		tables, err := parseNames(tc.tag)
		if tc.expectError {
			if err == nil {
				t.Errorf("parseDeclareTag(%q) = %v, want error", tc.tag, tables)
			}
		} else {
			if err != nil {
				t.Errorf("parseDeclareTag(%q) returned unexpected error: %v", tc.tag, err)
			} else {
				if len(tables) != len(tc.expectedTables) {
					t.Errorf("parseDeclareTag(%q) = %v, want %v", tc.tag, tables, tc.expectedTables)
					continue
				}
				for i := range tables {
					if tables[i] != tc.expectedTables[i] {
						t.Errorf("parseDeclareTag(%q) = %v, want %v", tc.tag, tables, tc.expectedTables)
						break
					}
				}
			}
		}
	}
}
