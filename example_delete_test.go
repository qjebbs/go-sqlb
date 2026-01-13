package sqlb_test

import (
	"context"
	"fmt"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
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
			"? NOT IN (?)",
			sqlf.Identifier("id"),
			sqlb.NewSelectBuilder().
				Select(bar.Column("foo_id")).
				From(bar),
		))
	ctx := sqlb.ContextWithDialect(context.Background(), dialect.SQLite{})
	query, args, err := b.Build(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// DELETE FROM "foo" WHERE "id" NOT IN (SELECT "b"."foo_id" FROM "bar" AS "b")
	// []
}
