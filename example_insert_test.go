package sqlb_test

import (
	"fmt"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

func ExampleInsertBuilder() {
	var foo = sqlb.NewTable("foo", "f")
	b := sqlb.NewInsertBuilder().
		InsertInto(foo).
		Columns("a", "b", "c").
		Values(1, 2, 3).
		Values(4, 5, 6).
		Returning("id")
	query, args, err := b.BuildQuery(sqlf.BindStyleQuestion)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// INSERT INTO foo (a, b, c) VALUES (?, ?, ?), (?, ?, ?) RETURNING id
	// [1 2 3 4 5 6]
}

func ExampleInsertBuilder_complex() {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar", "b")
	)
	b := sqlb.NewInsertBuilder().
		With(bar, sqlf.F("SELECT 1, 2")).
		InsertInto(foo).
		Columns("a", "b").
		From(
			sqlb.NewSelectBuilder().
				Select(bar.Column("*")).
				From(bar),
		).
		OnConflict("a").
		DoUpdateSet(sqlf.F("b = excluded.b")).
		Returning("id")
	query, args, err := b.BuildQuery(sqlf.BindStyleQuestion)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// WITH bar AS (SELECT 1, 2) INSERT INTO foo (a, b) SELECT b.* FROM bar AS b ON CONFLICT (a) DO UPDATE SET b = excluded.b RETURNING id
	// []
}
