package sqlb_test

import (
	"context"
	"testing"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlf/v4"
)

func TestBuildToContext(t *testing.T) {
	b := sqlf.F(
		"WHERE IN (?)",
		sqlb.NewSelectBuilder().
			Select(sqlf.Identifier("id")).
			From(sqlb.NewTable("foo")),
	)
	ctx := sqlb.NewContext(context.Background(), dialect.PostgreSQL{})
	query, args, err := b.Build(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantQuery := `WHERE IN (SELECT "id" FROM "foo")`
	if query != wantQuery {
		t.Fatalf("got query %q, want %q", query, wantQuery)
	}
	if len(args) != 0 {
		t.Fatalf("got args %v, want empty", args)
	}
}
