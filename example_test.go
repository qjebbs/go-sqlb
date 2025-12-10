package sqlb_test

import (
	"fmt"
	"time"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

func Example_elimination() {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar", "b")
		baz = sqlb.NewTable("baz", "z")
	)
	b := sqlb.NewQueryBuilder().
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
			"?=?",
			baz.Column("id"),
			foo.Column("baz_id"),
		)).
		Where(sqlf.F(
			"($2=$1 OR $3=$1)",
			1, foo.Column("a"), foo.Column("b"),
		)).
		Where2(bar.Column("c"), "=", 2)

	query, args, err := b.BuildQuery(sqlf.BindStyleQuestion)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// SELECT DISTINCT f.* FROM foo AS f INNER JOIN bar AS b ON b.foo_id=f.id WHERE (f.a=? OR f.b=?) AND b.c=?
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
	fooBuilder := sqlf.F("SELECT * FROM users WHERE active")
	barBuilder := sqlf.F("SELECT * FROM ?", foo) // requires 'foo'
	builder := sqlb.NewQueryBuilder().
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

func ExampleQueryBuilder_Debug() {
	foo := sqlb.NewTable("foo", "f")
	fooID := foo.Column("id")
	bar := sqlb.NewTable("bar", "b")
	cte := sqlb.NewTable("cte", "c")
	cteID := cte.Column("id")
	cteBuilder := sqlf.F("SELECT 1")
	q1 := sqlb.NewQueryBuilder().Debug("q1").
		Select(cteID).
		From(cte).
		InnerJoin(bar, sqlf.F("TRUE"))
	q2 := sqlb.NewQueryBuilder().Debug("q2").
		With(cte, cteBuilder).
		Select(cteID).
		From(cte).
		Union(q1)
	q3 := sqlb.NewQueryBuilder().Debug("q3").
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

func ExampleQueryStruct() {
	type Model struct {
		ID      int        `sqlb:"?.id"`
		Created *time.Time `sqlb:"?.created_at"`
		Updated *time.Time `sqlb:"?.updated_at"`
		Deleted *time.Time `sqlb:"?.deleted_at"`
	}

	type User struct {
		*Model `sqlb:"u"` // Anonymous field allows declaring tables for its fields
		Name   string     `sqlb:"u.name"`              // Simple syntax
		Age    int        `sqlb:"COALESCE(?.age,0);u"` // Equals to sqlf.F("COALESCE(?.age,0)", u)
	}

	Users := sqlb.NewTable("users", "u")
	b := sqlb.NewQueryBuilder().From(Users).Limit(10)
	b.Debug() // enable debug to see the built query
	defer func() {
		if err := recover(); err != nil {
			// ignore error since db is nil
		}
	}()
	sqlb.QueryStruct[*User](nil, b, sqlf.BindStyleDollar)
	// Output:
	// [sqlb] SELECT u.id, u.created_at, u.updated_at, u.deleted_at, u.name, COALESCE(u.age,0) FROM users AS u LIMIT 10
}
