package sqlb

import "testing"

func TestParseSimpleTag(t *testing.T) {
	testCases := []struct {
		tag            string
		expectedTable  string
		expectedColumn string
		expectedOk     bool
	}{
		{"u.id", "u", "id", true},
		{"user.name", "user", "name", true},
		{"a.b.c", "", "", false},         // multiple dots
		{"1table.column", "", "", false}, // starts with digit
		{"table.column$", "", "", false}, // invalid character
		{"tablecolumn", "", "", false},   // no dot
		{"table.", "", "", false},        // no column
		{"?.id", "", "", false},          // invalid character
	}

	for _, tc := range testCases {
		table, column, ok := parseSimpleTag(tc.tag)
		if table != tc.expectedTable || column != tc.expectedColumn || ok != tc.expectedOk {
			t.Errorf("parseSimpleTag(%q) = (%q, %q, %v); want (%q, %q, %v)",
				tc.tag, table, column, ok,
				tc.expectedTable, tc.expectedColumn, tc.expectedOk)
		}
	}
}

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
		tables, err := parseDeclareTag(tc.tag, 0)
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
