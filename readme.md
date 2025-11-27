## Go SQL QueryBuilder

Package sqlb provides a complex SQL query builder shipped  with WITH-CTE / JOIN 
Elimination capabilities, while [go-sqlf](https://github.com/qjebbs/go-sqlf) is the underlying foundation.

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
	// Output:
	// SELECT f.* FROM foo AS f INNER JOIN bar AS b ON b.foo_id=f.id WHERE (f.a=$1 OR f.b=$1) AND b.c=$2
	// [1 2]
}
```