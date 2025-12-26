# Go SQL Builder with Struct Mapping

sqlb is a powerful SQL builder and struct mapper. It provides,

- SQL builders to craft complex queries.
- Effortlessly map query results to Go structs.
- Declarative automation of common CRUD operations.

With sqlb, All queries are explicitly coded or declared, there is no hidden behavior, preserving both flexibility and transparency in your database interactions.

## Complex SELECT with Struct Mapping

sqlb provides a select builder shipped with WITH-CTE / JOIN
Elimination abilities, allowing you to build sophisticated SQL queries programmatically,
and easily map the results to nested structs with `mapper.Select()`.

```go
func Example_complexSelect() {
	type Model struct {
		ID      int        `sqlb:"col:id"`
		Created *time.Time `sqlb:"col:created"`
		Updated *time.Time `sqlb:"col:updated"`
		Deleted *time.Time `sqlb:"col:deleted"`
	}

	type User struct {
		// 'from' defines the from table in SQL for this struct,
		// it can be inherited by nested fields and by subsequent sibling fields of the current struct.
		Model `sqlb:"from:u"`
		// For fields without 'sel' tag, mapper constructs the selection column
		// from the 'from' tag (inherited here) and the 'col' tag of the field.
		// It is equivalent to:
		//  table := sqlb.NewTable("", "u")
		//  identifier := table.Column("name")
		//  const expr = "?.?"
		//  sel := sqlf.F(expr, table, identifier)
		Name string `sqlb:"col:name"`
	}

	type userListItem struct {
		User
		// OrgName is from another table and could be NULL,
		// 'sel' works together with 'from', which is equivalent to:
		//  table := sqlb.NewTable("", "o")
		//  expr := "COALESCE(?.name,'')"
		//  sel := sqlf.F(expr, table)
		OrgName string `sqlb:"sel:COALESCE(?.name,'');from:o"`
	}

	Users := sqlb.NewTable("users", "u")
	Orgs := sqlb.NewTable("orgs", "o")
	b := sqlb.NewSelectBuilder().
		From(Users).
		LeftJoin(Orgs, sqlf.F(
			"? = ?",
			Users.Column("org_id"),
			Orgs.Column("id"),
		)).
		WhereEquals(Orgs.Column("id"), 1).
		WhereIsNull(Users.Column("deleted_at"))
	_, err := mapper.Select[*userListItem](nil, b, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}
	// Output:
	// [Select(*mapper_test.userListItem)] SELECT u.id, u.created, u.updated, u.deleted, u.name, COALESCE(o.name,'') FROM users AS u LEFT JOIN orgs AS o ON u.org_id = o.id WHERE o.id = 1 AND u.deleted_at IS NULL
}
```

See Also:

- [example_select_test.go](./example_select_test.go) for more SELECT builder examples.


## CRUD Operations

sqlb provides declarative struct mapping via `mapper` package,

To avoid unnecessary abstraction, all CRUD operation inputs must represent explicit objects, not arbitrary query conditions. Therefore:

- Insert() supports batch insertion of multiple records, as each item is a concrete object to be inserted.
- Load(), Update(), and Delete() only operate on a single record at a time. If you need to update or delete multiple records in bulk, please use the sqlb package to construct the corresponding SQL statements manually.

```go

import (
	"errors"
	"fmt"
	"time"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/mapper"
	"github.com/qjebbs/go-sqlf/v4"
)

func Example_cRUD() {
	// Model represents the base model with common fields.
	type Model struct {
		// ID is the model ID.
		// pk means primary key, which is used to locate records for Load / Update / Delete operations.
		// returning means the ID will be returned after insertion.
		ID int64 `sqlb:"col:id;pk;returning"`
		// Created is the creation time.
		// readonly means this field will be excluded from INSERT / UPDATE, created usually set by DB default value.
		Created *time.Time `sqlb:"col:created;readonly"`
		// Updated is the last update time.
		// conflict_set means when inserting an existing record, the Updated will be updated.
		Updated *time.Time `sqlb:"col:updated;conflict_set"`
		// soft_delete indicates this column is used for soft deletion.
		// When deleting, the column will be set to current time or true instead of actually deleting the record.
		// When loading, records with non-zero value in this column will be ignored.
		// conflict_set means when inserting an deleted record, undelete it.
		Deleted *time.Time `sqlb:"col:deleted;soft_delete;conflict_set"`
	}

	type User struct {
		// The value defined by 'tables' and 'from' can be inherited by nested fields
		// and by subsequent sibling fields of the current struct.
		Model `sqlb:"table:users"`
		// unique indicates this column has a unique constraint, which can be used to locate records for Load / Delete operation.
		// conflict_on indicates the column(s) to check for conflict during insert.
		Email string `sqlb:"col:email;required;unique;conflict_on"`
		// conflict_set without value means to use excluded column value
		Name string `sqlb:"col:name;conflict_set"`
	}

	err := mapper.Insert(nil, []*User{
		{Email: "alice@example.org", Name: "Alice"},
		{Email: "bob@example.org", Name: ""},
	}, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}

	_, err = mapper.Load(nil, &User{Model: Model{ID: 1}}, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}
	_, err = mapper.Exists(nil, &User{Model: Model{ID: 1}}, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}

	// Partial update: only non-zero fields will be updated.
	// Here we located the record by unique Email field.
	err = mapper.Patch(nil, &User{
		Email: "alice@example.org",
		Name:  "Happy Alice",
	}, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}

	err = mapper.Update(nil, &User{Email: "alice@example.org"}, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}

	// You don't have to set the .Deleted field manually, mapper will set it to current time automatically.
	// But here we set it to a fixed value to make the example output deterministic.
	user := &User{Model: Model{ID: 1}}
	user.Deleted = &time.Time{}
	err = mapper.Delete(nil, user, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}
	// Output:
	// [Insert(*mapper_test.User)] INSERT INTO users (email, name) VALUES ('alice@example.org', 'Alice'), ('bob@example.org', DEFAULT) ON CONFLICT (email) DO UPDATE SET updated = EXCLUDED.updated, deleted = EXCLUDED.deleted, name = EXCLUDED.name RETURNING id
	// [Load(*mapper_test.User)] SELECT created, updated, email, name FROM users WHERE id = 1 AND deleted IS NULL
	// [Exists(*mapper_test.User)] SELECT 1 FROM users WHERE id = 1 AND deleted IS NULL
	// [Patch(*mapper_test.User)] UPDATE users SET name = 'Happy Alice' WHERE email = 'alice@example.org' AND deleted IS NULL
	// [Update(*mapper_test.User)] UPDATE users SET updated = NULL, name = '' WHERE email = 'alice@example.org' AND deleted IS NULL
	// [Delete(*mapper_test.User)] UPDATE users SET deleted = '0001-01-01 00:00:00 +0000 UTC' WHERE id = 1 AND deleted IS NULL
}
```
