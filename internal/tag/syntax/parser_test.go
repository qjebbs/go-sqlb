package syntax

import (
	"reflect"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	emptyStr := ""
	exprStr := "expr"
	testCases := []struct {
		raw     string
		want    *Info
		wantErr bool
	}{
		{
			raw:     ":a",
			wantErr: true,
		},
		{
			raw:     "sel:;",
			wantErr: true,
		},
		// string pointer values
		{
			raw: "conflict_set;",
			want: &Info{
				ConflictSet: &emptyStr,
			},
		},
		{
			raw: "conflict_set:expr;",
			want: &Info{
				ConflictSet: &exprStr,
			},
		},
		// string values
		{
			raw: "col:id;",
			want: &Info{
				Column: "id",
			},
		},
		{
			raw: "sel:?.id;from:u;",
			want: &Info{
				Select: "?.id",
				From:   []string{"u"},
			},
		},
		{
			raw: "sel:COALESCE(?.age,0);from:u;",
			want: &Info{
				Select: "COALESCE(?.age,0)",
				From:   []string{"u"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := Parse(tc.raw)
			if !tc.wantErr && err != nil {
				t.Fatal(err)
			}
			if !tc.wantErr {
				if got.Column != tc.want.Column {
					t.Errorf("got Column %q, want %#q", got.Column, tc.want.Column)
				}
				if !reflect.DeepEqual(got.From, tc.want.From) {
					t.Errorf("got Tables %q, want %q", strings.Join(got.From, ","), strings.Join(tc.want.From, ","))
				}
				if !reflect.DeepEqual(got.On, tc.want.On) {
					t.Errorf("got On %q, want %q", strings.Join(got.On, ","), strings.Join(tc.want.On, ","))
				}
			}
		})
	}
}
