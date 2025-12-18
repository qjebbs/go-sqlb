## Go SQL Builder with Struct Mapping

Package sqlb provides a complex SQL query builder shipped with WITH-CTE / JOIN 
Elimination abilities, and struct mapping capabilities (by `mapper` package), while [go-sqlf](https://github.com/qjebbs/go-sqlf) is the underlying foundation.

See Also:

- [example_test.go](./example_test.go) for more builder examples.
- [mapper/example_test.go](./mapper/example_test.go) for more mapping examples.
- [syntax.md](./mapper/syntax.md) for mapping syntax.

```go
import (
	"time"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/mapper"
)

func ExampleSelect() {
	type Model struct {
		ID      int        `sqlb:"col:id"`
		Created *time.Time `sqlb:"col:created"`
		Updated *time.Time `sqlb:"col:updated"`
		Deleted *time.Time `sqlb:"col:deleted"`
	}

	type User struct {
		Model `sqlb:"from:u"`
		Name  string `sqlb:"col:name"`
	}

	Users := sqlb.NewTable("users", "u")
	b := sqlb.NewSelectBuilder().
		From(Users).
		WhereEquals(Users.Column("id"), 1)
	// Will build following query, and map the result into []*User
	// SELECT u.id, u.created, u.updated, u.deleted, u.name FROM users AS u WHERE u.id = 1
	users, err := mapper.Select[*User](db, b)
}
```