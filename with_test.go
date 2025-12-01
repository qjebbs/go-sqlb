package sqlb

import (
	"fmt"
	"sort"
	"testing"

	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

func TestWithElimination(t *testing.T) {
	foo := NewTable("foo")
	bar := NewTable("bar")
	query, _, err := _With(foo, sqlf.F("SELECT 1")).
		With(bar, sqlf.F("SELECT 2")).
		For(sqlf.F(
			"SELECT * FROM ?", foo,
		)).
		BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	// Output:
	// WITH foo AS (SELECT 1) SELECT * FROM foo
}

func TestWithEmptyCTE(t *testing.T) {
	w := newCTEs().For(sqlf.F("SELECT 1"))
	query, err := w.Build(sqlf.NewContext(sqlf.BindStyleQuestion))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "SELECT 1"
	if query != expected {
		t.Fatalf("expected %q, got %q", expected, query)
	}
}

func TestWithEmptyFor(t *testing.T) {
	w := _With(NewTable("foo"), sqlf.F("SELECT 1"))
	query, err := w.Build(sqlf.NewContext(sqlf.BindStyleQuestion))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := ""
	if query != expected {
		t.Fatalf("expected %q, got %q", expected, query)
	}
}

func TestWithReportDeps(t *testing.T) {
	foo := NewTable("foo")
	bar := NewTable("bar")
	w := _With(foo, sqlf.F("SELECT 1")).
		For(sqlf.F(
			"SELECT * FROM ? INNER JOIN ?", foo, bar,
		))
	deps := newDepTables()
	ctx := contextWithDepTables(sqlf.NewContext(sqlf.BindStyleQuestion), deps)
	_, err := w.Build(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With only reports unresolved deps to parent's decendent deps.
	wantDecendent := []string{"bar"}

	got := util.Map(util.MapKeys(deps.tables), func(t Table) string { return t.Name })
	assertDeps(t, wantDecendent, got)
}

func assertDeps(t *testing.T, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("expected deps %v, got %v", want, got)
	}
	sort.Slice(want, func(i, j int) bool {
		return want[i] < want[j]
	})
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("expected deps %v, got %v", want, got)
		}
	}
}
