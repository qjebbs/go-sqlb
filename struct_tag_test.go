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
