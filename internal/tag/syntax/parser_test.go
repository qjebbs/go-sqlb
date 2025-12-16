package syntax

import (
	"reflect"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
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
			raw: "col:id;",
			want: &Info{
				Column: "id",
			},
		},
		{
			raw: "sel:?.id;tables:u;",
			want: &Info{
				Select: "?.id",
				Tables: []string{"u"},
			},
		},
		{
			raw:  "sel:;",
			want: &Info{},
		},
		{
			raw: "sel:COALESCE(?.age,0);tables:u;",
			want: &Info{
				Select: "COALESCE(?.age,0)",
				Tables: []string{"u"},
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
				if !reflect.DeepEqual(got.Tables, tc.want.Tables) {
					t.Errorf("got Tables %q, want %q", strings.Join(got.Tables, ","), strings.Join(tc.want.Tables, ","))
				}
				if !reflect.DeepEqual(got.On, tc.want.On) {
					t.Errorf("got On %q, want %q", strings.Join(got.On, ","), strings.Join(tc.want.On, ","))
				}
			}
		})
	}
}
