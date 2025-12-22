package sqlb_test

import (
	"fmt"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

func ExampleDeleteBuilder_postgreSQL() {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar", "b")
	)
	// delete from foo where have no matching records in bar
	b := sqlb.NewDeleteBuilder().
		DeleteFrom(foo.Name).
		Where(sqlf.F(
			"id NOT IN (?)",
			sqlb.NewSelectBuilder().
				Select(bar.Column("foo_id")).
				From(bar),
		))
	query, args, err := b.BuildQuery(sqlf.BindStyleQuestion)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// DELETE FROM foo WHERE id NOT IN (SELECT b.foo_id FROM bar AS b)
	// []
}
