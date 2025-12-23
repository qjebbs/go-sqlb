package mapper_test

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
	}

	type User struct {
		// The value defined by 'tables' and 'from' can be inherited by nested fields
		// and by subsequent sibling fields of the current struct.
		Model `sqlb:"table:users"`
		// unique indicates this column has a unique constraint, which can be used to locate records for Load / Delete operation.
		// conflict_on indicates the column(s) to check for conflict during insert.
		Email string `sqlb:"col:email;unique;conflict_on"`
		// conflict_set without value means to use excluded column value
		Name      string `sqlb:"col:name;conflict_set"`
		LoginName string `sqlb:"col:login_name;load:COALESCE(?,'');"`
	}

	user := &User{Email: "alice@example.org", Name: "Alice"}
	user2 := &User{Email: "bob@example.org", Name: ""}
	err := mapper.Insert(nil, []*User{user, user2}, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}
	// After insertion, the ID field will be populated and set by mapper.
	// Here we just simulate it.
	user.ID = 1
	user2.ID = 2

	_, err = mapper.Load(nil, &User{Email: "alice@example.org"}, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}

	// Partial update: only non-zero fields will be updated.
	err = mapper.Update(nil, &User{
		Model: Model{ID: user.ID},
		Name:  "Alice",
	}, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}

	user.Name = ""
	err = mapper.Update(nil, user, mapper.WithUpdateAll(), mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}

	_, err = mapper.Delete(nil, user, mapper.WithDebug())
	if err != nil && !errors.Is(err, mapper.ErrNilDB) {
		fmt.Println(err)
	}
	// Output:
	// [Insert(*mapper_test.User)] INSERT INTO users (email, name) VALUES ('alice@example.org', 'Alice'), ('bob@example.org', DEFAULT) ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name RETURNING id
	// [Load(*mapper_test.User)] SELECT created, updated, name, COALESCE(login_name,''), id FROM users WHERE email = 'alice@example.org'
	// [Update(*mapper_test.User)] UPDATE users SET name = 'Alice' WHERE id = 1
	// [Update(*mapper_test.User)] UPDATE users SET updated = NULL, email = 'alice@example.org', name = '', login_name = '' WHERE id = 1
	// [Delete(*mapper_test.User)] DELETE FROM users WHERE id = 1
}

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
