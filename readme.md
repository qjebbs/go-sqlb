## Go SQL Builder with Struct Mapping

This package champions a declarative, "definition as documentation" philosophy for database interactions in Go. It empowers developers to define data models and their corresponding database behaviors directly within Go structs using sqlb tags, abstracting away the need for manual SQL writing for standard CRUD operations.

> sqlb utilizes [go-sqlf](https://github.com/qjebbs/go-sqlf) as the underlying foundation.

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
		// noupdate means this field will not be updated on update operations.
		Created *time.Time `sqlb:"col:created;noupdate"`
		// Updated is the last update time.
		// conflict_set means when inserting an existing record, the Updated will be updated.
		Updated *time.Time `sqlb:"col:updated;conflict_set"`
	}

	type User struct {
		// The value defined by 'tables' can be inherited by nested fields
		// and by subsequent sibling fields of the current struct.
		Model `sqlb:"table:users"`
		// unique indicates this column has a unique constraint, which can be used to locate records for Load / Delete operation.
		// conflict_on indicates the column(s) to check for conflict during insert.
		Email string `sqlb:"col:email;unique;conflict_on"`
		// conflict_set without value means to use excluded column value
		Name string `sqlb:"col:name;conflict_set"`
		// Included only when the "full" tag is specified
		// conflict_set can accept SQL expressions
		About string `sqlb:"col:about;on:full;conflict_set:CASE WHEN excluded.about = '' THEN users.about ELSE excluded.about END"`
	}

	created := time.Date(2020, 1, 1, 10, 10, 0, 0, time.UTC)
	update := created.Add(time.Hour)

	user := &User{Email: "alice@example.org"}
	user.Created = &created
	err := mapper.InsertOne(nil, user, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}
	// After insertion, the ID field will be populated and set by mapper.
	// Here we just simulate it.
	user.ID = 1

	_, err = mapper.Load(nil, &User{Email: "alice@example.org"}, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}

	user.Updated = &update
	user.Name = "New Name"
	err = mapper.Update(nil, user, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}

	_, err = mapper.Delete(nil, user, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}
	// Output:
	// [Insert(*mapper_test.User)] INSERT INTO users (created, updated, email, name, about) VALUES ('2020-01-01 10:10:00 +0000 UTC', NULL, 'alice@example.org', '', '') ON CONFLICT (email) DO UPDATE SET updated = EXCLUDED.updated, name = EXCLUDED.name, about = CASE WHEN excluded.about = '' THEN users.about ELSE excluded.about END RETURNING id
	// [Load(*mapper_test.User)] SELECT created, updated, name, about, id FROM users WHERE email = 'alice@example.org'
	// [Update(*mapper_test.User)] UPDATE users SET updated = '2020-01-01T11:10:00Z', email = 'alice@example.org', name = 'New Name', about = '' WHERE id = 1
	// [Delete(*mapper_test.User)] DELETE FROM users WHERE id = 1
}

```

For complex queries, sqlb provides a select builder shipped with WITH-CTE / JOIN 
Elimination abilities, allowing you to build sophisticated SQL queries programmatically.

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
		// Dive into User struct to include its fields
		User `sqlb:"dive"`
		// OrgName is from joined table,
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
		WhereEquals(Orgs.Column("id"), 1)
	_, err := mapper.Select[*userListItem](nil, b, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}
	// Output:
	// [Select(*mapper_test.userListItem)] SELECT u.id, u.created, u.updated, u.deleted, u.name, COALESCE(o.name,'') FROM users AS u LEFT JOIN orgs AS o ON u.org_id = o.id WHERE o.id = 1
}
```

See Also:

- [example_select_test.go](./example_select_test.go) for more SELECT builder examples.
- [example_update_test.go](./example_update_test.go) for more UPDATE operation examples.
