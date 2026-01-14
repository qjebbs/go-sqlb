package sqlb_test

import (
	"context"
	"fmt"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlf/v4"
)

func ExampleInsertBuilder() {
	b := sqlb.NewInsertBuilder().
		InsertInto("foo").
		Columns("a", "b", "c").
		Values(1, 2, 3).
		Values(4, 5, 6).
		Returning("id")
	ctx := sqlb.NewContext(context.Background(), dialect.SQLite{})
	query, args, err := b.Build(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// INSERT INTO "foo" ("a", "b", "c") VALUES (?, ?, ?), (?, ?, ?) RETURNING "id"
	// [1 2 3 4 5 6]
}

func ExampleInsertBuilder_complex() {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar", "b")
	)
	b := sqlb.NewInsertBuilder().
		With(bar, sqlf.F("SELECT 1, 2")).
		InsertInto(foo.Name).
		Columns("a", "b").
		From(
			sqlb.NewSelectBuilder().
				Select(bar.AllColumns()).
				From(bar),
		).
		OnConflict([]string{"a"}, sqlf.F("$1 = EXCLUDED.$1", sqlf.Identifier("b"))).
		Returning("id")
	ctx := sqlb.NewContext(context.Background(), dialect.PostgreSQL{})
	query, args, err := b.Build(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// WITH "bar" AS (SELECT 1, 2) INSERT INTO "foo" ("a", "b") SELECT "b".* FROM "bar" AS "b" ON CONFLICT ("a") DO UPDATE SET "b" = EXCLUDED."b" RETURNING "id"
	// []
}
