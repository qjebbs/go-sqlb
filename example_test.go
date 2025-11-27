package sqlb_test

import (
	"fmt"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

func ExampleQueryBuilder_BuildQuery() {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar", "b")
	)
	b := sqlb.NewQueryBuilder().
		Select(foo.Column("*")).
		From(foo).
		InnerJoin(bar, sqlf.F(
			"?=?",
			bar.Column("foo_id"),
			foo.Column("id"),
		)).
		Where(sqlf.F(
			"($2=$1 OR $3=$1)",
			1, foo.Column("a"), foo.Column("b"),
		)).
		Where2(bar.Column("c"), "=", 2)

	query, args, err := b.BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	query, args, err = b.BuildQuery(sqlf.BindStyleQuestion)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// SELECT f.* FROM foo AS f INNER JOIN bar AS b ON b.foo_id=f.id WHERE (f.a=$1 OR f.b=$1) AND b.c=$2
	// [1 2]
	// SELECT f.* FROM foo AS f INNER JOIN bar AS b ON b.foo_id=f.id WHERE (f.a=? OR f.b=?) AND b.c=?
	// [1 1 2]
}

func ExampleQueryBuilder_LeftJoinOptional() {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar", "b")
	)
	query, args, err := sqlb.NewQueryBuilder().
		Distinct(). // *QueryBuilder eliminates optional joins when SELECT DISTINCT is used.
		Select(foo.Column("*")).
		From(foo).
		// declare an optional LEFT JOIN
		LeftJoinOptional(bar, sqlf.F(
			"?=?",
			bar.Column("foo_id"),
			foo.Column("id"),
		)).
		// don't touch any columns of "bar", so that it can be eliminated
		Where2(foo.Column("id"), ">", 1).
		BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// SELECT DISTINCT f.* FROM foo AS f WHERE f.id>$1
	// [1]
}

func ExampleQueryBuilder_With() {
	foo := sqlb.NewTable("foo")
	bar := sqlb.NewTable("bar")
	builderFoo := sqlf.F("SELECT * FROM users WHERE active")
	builderBar := sqlf.F("SELECT * FROM ?", foo) // requires 'foo'
	builder := sqlb.NewQueryBuilder().
		With(foo, builderFoo).
		With(bar, builderBar).
		Select(bar.Column("*")). // requires 'bar'
		From(bar)                // requires 'bar'

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
	// With foo AS (SELECT * FROM users WHERE active), bar AS (SELECT * FROM foo) SELECT bar.* FROM bar
}

func ExampleQueryBuilder_Union() {
	var foo = sqlb.NewTable("foo", "f")
	column := foo.Column("*")
	query, args, err := sqlb.NewQueryBuilder().
		Select(column).
		From(foo).
		Where2(foo.Column("id"), " = ", 1).
		Union(
			sqlb.NewQueryBuilder().
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

func ExampleNoDeps() {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar")
	)
	q := sqlb.NewQueryBuilder().
		Select(foo.Column("bar")).
		From(foo).
		Where(
			// will not report 'b' as dependency, or the builder complains: from undefined: 'bar'
			sqlf.F(
				"? IN (?)",
				foo.Column("id"),
				sqlb.NoDeps(sqlf.F(
					"SELECT id FROM ?", bar,
				)),
			),
		)
	query, args, err := q.BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// SELECT f.bar FROM foo AS f WHERE f.id IN (SELECT id FROM bar)
	// []
}

func ExampleQueryBuilder_Debug() {
	foo := sqlb.NewTable("foo", "f")
	id := foo.Column("id")
	q1 := sqlb.NewQueryBuilder().Debug("q1").
		Select(id).
		From(foo)
	q2 := sqlb.NewQueryBuilder().Debug("q2").
		Select(id).
		From(foo).
		Where(sqlf.F("? IN (?)", id, q1))
	q3 := sqlb.NewQueryBuilder().Debug("q3").
		Select(id).
		From(foo).
		Where(sqlf.F("? IN (?)", id, q2))
	q4 := sqlb.NewQueryBuilder().Debug("q4").
		Select(foo.Column("*")).
		From(foo).
		Where(sqlf.F("? IN (?)", id, q3))
	_, _, err := q4.BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Output:
	// [q1] SELECT f.id FROM foo AS f
	// [q2] SELECT f.id FROM foo AS f WHERE f.id IN (SELECT f.id FROM foo AS f)
	// [q3] SELECT f.id FROM foo AS f WHERE f.id IN (SELECT f.id FROM foo AS f WHERE f.id IN (SELECT f.id FROM foo AS f))
	// [q4] SELECT f.* FROM foo AS f WHERE f.id IN (SELECT f.id FROM foo AS f WHERE f.id IN (SELECT f.id FROM foo AS f WHERE f.id IN (SELECT f.id FROM foo AS f)))
}
