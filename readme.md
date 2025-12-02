## Go SQL QueryBuilder

Package sqlb provides a complex SQL query builder shipped with WITH-CTE / JOIN 
Elimination abilities, while [go-sqlf](https://github.com/qjebbs/go-sqlf) is the underlying foundation.

See [example_test.go](./example_test.go) for examples.

```go
import (
	"fmt"
	"github.com/qjebbs/go-sqlb"
)
func ExampleQueryBuilder_BuildQuery() {
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
		// Will be eliminated since no columns from "baz" are used.
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
```