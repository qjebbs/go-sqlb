package sqlb_test

import (
	"fmt"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/dialects"
	"github.com/qjebbs/go-sqlf/v4"
)

func ExampleUpdateBuilder_postgreSQL() {
	var (
		foo = sqlb.NewTable("foo")
		bar = sqlb.NewTable("bar", "b")
		baz = sqlb.NewTable("baz", "z")
	)
	b := sqlb.NewUpdateBuilder().
		Update(foo.AppliedName()).
		Set("a", 1).
		Set("baz", bar.Column("baz")).
		From(bar).
		InnerJoin(baz, sqlf.F(
			"? = ?", bar.Column("id"), baz.Column("bar_id"),
		)).
		WhereEquals(foo.Column("id"), bar.Column("foo_id"))
	query, args, err := b.BuildQuery(sqlf.BindStyleQuestion)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// UPDATE foo SET a = ?, baz = b.baz FROM bar AS b INNER JOIN baz AS z ON b.id = z.bar_id WHERE foo.id = b.foo_id
	// [1]
}

func ExampleUpdateBuilder_sqlServer() {
	var (
		// SQL Server UPDATE FROM does not support table aliasing
		foo = sqlb.NewTable("foo")
		bar = sqlb.NewTable("bar")
	)
	b := sqlb.NewUpdateBuilder(dialects.DialectSQLServer).
		Update(foo.Name).
		Set("a", 1).
		Set("baz", bar.Column("baz")).
		From(foo).
		InnerJoin(bar, sqlf.F("? = ?", foo.Column("id"), bar.Column("foo_id")))
	query, args, err := b.BuildQuery(sqlf.BindStyleQuestion)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// UPDATE foo SET a = ?, baz = bar.baz FROM foo INNER JOIN bar ON foo.id = bar.foo_id
	// [1]
}

func ExampleUpdateBuilder_mysql() {
	var (
		foo = sqlb.NewTable("foo")
		bar = sqlb.NewTable("bar", "z")
	)
	b := sqlb.NewUpdateBuilder(dialects.DialectMySQL).
		Update(foo.Name).
		Set("a", 1).
		Set("baz", bar.Column("baz")).
		InnerJoin(bar, sqlf.F("? = ?", foo.Column("id"), bar.Column("foo_id")))
	query, args, err := b.BuildQuery(sqlf.BindStyleQuestion)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// UPDATE foo INNER JOIN bar AS z ON foo.id = z.foo_id SET a = ?, baz = z.baz
	// [1]
}
