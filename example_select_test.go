package sqlb_test

import (
	"fmt"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

func Example_elimination() {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar", "b")
		baz = sqlb.NewTable("baz", "z")
	)
	b := sqlb.NewSelectBuilder().
		EnableElimination().
		// Will be eliminated since not required.
		With(baz, sqlf.F("SELECT 1")).
		Distinct().Select(foo.Column("*")).
		From(foo).
		InnerJoin(bar, sqlf.F(
			"?=?",
			bar.Column("foo_id"),
			foo.Column("id"),
		)).
		// Will be eliminated since SELECT DISTINCT and no columns from "baz" are used.
		LeftJoinOptional(baz, sqlf.F(
			"? = ?",
			baz.Column("id"),
			foo.Column("baz_id"),
		)).
		Where(sqlf.F(
			"($2 = $1 OR $3 = $1)",
			1, foo.Column("a"), foo.Column("b"),
		)).
		WhereEquals(bar.Column("c"), 2)

	query, args, err := b.BuildQuery(sqlf.BindStyleQuestion)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// SELECT DISTINCT f.* FROM foo AS f INNER JOIN bar AS b ON b.foo_id=f.id WHERE (f.a = ? OR f.b = ?) AND b.c = ?
	// [1 1 2]
}

func ExampleSelectBuilder_LeftJoinOptional() {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar", "b")
	)
	query, args, err := sqlb.NewSelectBuilder().
		EnableElimination().
		Distinct(). // *SelectBuilder eliminates optional joins when SELECT DISTINCT is used.
		Select(foo.Column("*")).
		From(foo).
		// declare an optional LEFT JOIN
		LeftJoinOptional(bar, sqlf.F(
			"? = ?",
			bar.Column("foo_id"),
			foo.Column("id"),
		)).
		// don't touch any columns of "bar", so that it can be eliminated
		WhereGreaterThan(foo.Column("id"), 1).
		BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// SELECT DISTINCT f.* FROM foo AS f WHERE f.id > $1
	// [1]
}

func ExampleSelectBuilder_With() {
	foo := sqlb.NewTable("foo")
	bar := sqlb.NewTable("bar")
	fooBuilder := sqlf.F("SELECT * FROM users WHERE active")
	barBuilder := sqlf.F("SELECT * FROM ?", foo) // requires 'foo'
	builder := sqlb.NewSelectBuilder().
		EnableElimination().
		With(foo, fooBuilder).
		With(bar, barBuilder).
		Select(bar.Column("*")). // requires 'bar'
		From(bar)

	// Tracked dependencies:
	// - SELECT / FROM requires 'bar',
	// - 'bar' requires 'foo',
	// so both CTEs are included.
	query, _, err := builder.BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	// Output:
	// WITH foo AS (SELECT * FROM users WHERE active), bar AS (SELECT * FROM foo) SELECT bar.* FROM bar
}

func ExampleSelectBuilder_Union() {
	var foo = sqlb.NewTable("foo", "f")
	column := foo.Column("*")
	query, args, err := sqlb.NewSelectBuilder().
		Select(column).
		From(foo).
		WhereEquals(foo.Column("id"), 1).
		Union(
			sqlb.NewSelectBuilder().
				From(foo).
				WhereIn(foo.Column("id"), []any{2, 3, 4}).
				Select(column),
		).
		BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// SELECT f.* FROM foo AS f WHERE f.id = $1 UNION SELECT f.* FROM foo AS f WHERE f.id IN ($2, $3, $4)
	// [1 2 3 4]
}

func ExampleSelectBuilder_Debug() {
	foo := sqlb.NewTable("foo", "f")
	fooID := foo.Column("id")
	bar := sqlb.NewTable("bar", "b")
	cte := sqlb.NewTable("cte", "c")
	cteID := cte.Column("id")
	cteBuilder := sqlf.F("SELECT 1")
	q1 := sqlb.NewSelectBuilder().Debug("q1").
		Select(cteID).
		From(cte).
		InnerJoin(bar, sqlf.F("TRUE"))
	q2 := sqlb.NewSelectBuilder().Debug("q2").
		With(cte, cteBuilder).
		Select(cteID).
		From(cte).
		Union(q1)
	q3 := sqlb.NewSelectBuilder().Debug("q3").
		EnableElimination().
		With(cte, cteBuilder).
		Select(foo.Column("*")).
		From(foo).
		Where(sqlf.F("? IN (?)", fooID, q2))
	_, _, err := q3.BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Output:
	// [q1] SELECT c.id FROM cte AS c INNER JOIN bar AS b ON TRUE
	// [q2] WITH cte AS (SELECT 1) SELECT c.id FROM cte AS c UNION SELECT c.id FROM cte AS c INNER JOIN bar AS b ON TRUE
	// [q3] SELECT f.* FROM foo AS f WHERE f.id IN (WITH cte AS (SELECT 1) SELECT c.id FROM cte AS c UNION SELECT c.id FROM cte AS c INNER JOIN bar AS b ON TRUE)
}
