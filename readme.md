
# Go SQL Builder (`sqlb`)

**sqlb** is a powerful, flexible SQL builder for Go.
It helps you programmatically construct complex SQL queries with full transparency and zero hidden behavior.

## Features

- Chainable, composable SQL builders for SELECT, INSERT, UPDATE, DELETE, and more
- Support for advanced SQL features: WITH-CTE, JOIN, subqueries, expressions, etc.
- Full control over query structure, no hidden magic, no forced conventions
- Works seamlessly with any database/sql driver

## Example: Building a Complex SELECT

```go
import (
	"context"
	"fmt"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
)

func Example_select() {
	Users := sqlb.NewTable("users", "u")
	b := sqlb.NewSelectBuilder().
		Select(Users.Column("*")).
		From(Users).
		WhereEquals(Users.Column("org_id"), 1).
		WhereIsNull(Users.Column("deleted_at"))
	ctx := sqlb.NewContext(context.Background(), dialect.PostgreSQL{})
	query, args, err := b.Build(ctx)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(query)
	fmt.Println(args)
	// Output:
	// SELECT "u".* FROM "users" AS "u" WHERE "u"."org_id" = $1 AND "u"."deleted_at" IS NULL
	// [1]
}
```

See Also:

- [example_select_test.go](./example_select_test.go) for more SELECT builder examples.

## About Struct Mapping

> **Note:** Struct mapping and CRUD automation features have been moved to the [sqlm](https://github.com/qjebbs/sqlm) project.
> `sqlb` now focuses solely on SQL query building.

## The Go SQL Tools Family

This project is part of a family of Go SQL tools, each designed for a different level of abstraction and automation:

1. **[go-sqlf](https://github.com/qjebbs/go-sqlf)** — Minimalist SQL fragment builder. For simple, manual SQL composition with parameter binding and zero magic.
2. **go-sqlb** (this project) — Advanced SQL builder. For programmatically building complex queries (CTE, JOIN, expressions, etc.) with chainable, declarative, and composable APIs.
3. **[go-sqlm](https://github.com/qjebbs/sqlm)** — Struct mapping. Declarative struct mapping, automatic CRUD, batch operations, and high-performance zero-reflection code generation.