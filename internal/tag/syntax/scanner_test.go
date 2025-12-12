package syntax

import (
	"reflect"
	"testing"
)

func TestScanner(t *testing.T) {
	testCases := []struct {
		raw  string
		want []token
	}{
		{
			raw: "a:b",
			want: []token{
				{typ: _Key, lit: "a", bad: false, start: 0, end: 1},
				{typ: _Colon, lit: ":", bad: false, start: 1, end: 2},
				{typ: _Value, lit: "b", bad: false, start: 2, end: 3},
				{typ: _EOF, lit: "", bad: false, start: 3, end: 3},
			},
		},
		{
			raw: " a : b ;",
			want: []token{
				{typ: _Key, lit: "a", bad: false, start: 1, end: 2},
				{typ: _Colon, lit: ":", bad: false, start: 3, end: 4},
				{typ: _Value, lit: "b", bad: false, start: 5, end: 6},
				{typ: _Semicolon, lit: ";", bad: false, start: 7, end: 8},
				{typ: _EOF, lit: "", bad: false, start: 8, end: 8},
			},
		},
		{
			raw: "a::b",
			want: []token{
				{typ: _Key, lit: "a", bad: false, start: 0, end: 1},
				{typ: _Colon, lit: ":", bad: false, start: 1, end: 2},
				{typ: _Value, lit: ":b", bad: false, start: 2, end: 4},
				{typ: _EOF, lit: "", bad: false, start: 4, end: 4},
			},
		},
		{
			raw: "a::b;",
			want: []token{
				{typ: _Key, lit: "a", bad: false, start: 0, end: 1},
				{typ: _Colon, lit: ":", bad: false, start: 1, end: 2},
				{typ: _Value, lit: ":b", bad: false, start: 2, end: 4},
				{typ: _Semicolon, lit: ";", bad: false, start: 4, end: 5},
				{typ: _EOF, lit: "", bad: false, start: 5, end: 5},
			},
		},
		{
			raw: "a:;",
			want: []token{
				{typ: _Key, lit: "a", bad: false, start: 0, end: 1},
				{typ: _Colon, lit: ":", bad: false, start: 1, end: 2},
				{typ: _Semicolon, lit: ";", bad: false, start: 2, end: 3},
				{typ: _EOF, lit: "", bad: false, start: 3, end: 3},
			},
		},
		{
			raw: "::b;",
			want: []token{
				{typ: _Error, lit: ":", bad: true, start: 0, end: 1},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.raw, func(t *testing.T) {
			got := make([]token, 0)
			s := newScanner(tc.raw)
			for s.NextToken() {
				// ignore Pos
				s.token.pos = Pos{}
				got = append(got, *s.token)
			}
			if !reflect.DeepEqual(got, tc.want) {
				for _, tk := range got {
					t.Logf("%#v", tk)
				}
				t.Fatal("failed")
			}
		})
	}
}
