package mapper_test

import (
	"fmt"
	"time"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/mapper"
	"github.com/qjebbs/go-sqlf/v4"
)

func ExampleSelect() {
	type Model struct {
		ID      int        `sqlb:"col:id"`
		Created *time.Time `sqlb:"col:created_at"`
		Updated *time.Time `sqlb:"col:updated_at"`
		Deleted *time.Time `sqlb:"col:deleted_at"`
	}

	type User struct {
		// Anonymous field supports only the 'tables' key.
		// The value defined by 'tables' can be inherited by nested fields
		// and by subsequent sibling fields of the current struct.
		Model `sqlb:"table:users;from:u"`
		Name  string `sqlb:"col:name"`
		// Included only when the "full" tag is specified
		Notes string `sqlb:"col:notes;on:full"`
	}

	type UserOrg struct {
		// Dive into structs
		User *User `sqlb:"dive"`
		// Unlike col tags, a sel tag semantically suggest that
		// it's a SELECT expression rather than a column.
		OrgName float64 `sqlb:"sel:COALESCE(?.name,'');from:o"`
	}

	Users := sqlb.NewTable("users", "u")
	Orgs := sqlb.NewTable("orgs", "o")
	b := sqlb.NewSelectBuilder().
		From(Users).
		LeftJoin(Orgs, sqlf.F(
			"?.org_id = ?.id",
			Users, Orgs,
		)).
		WhereEquals(Users.Column("id"), 1)
	b.Debug() // enable debug to see the built query
	defer func() {
		if err := recover(); err != nil {
			// ignore error since db is nil
		}
	}()
	_, err := mapper.Select[*UserOrg](nil, b)
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// [sqlb] SELECT u.id, u.created_at, u.updated_at, u.deleted_at, u.name, COALESCE(o.name,'') FROM users AS u LEFT JOIN orgs AS o ON u.org_id = o.id WHERE u.id = 1
}

func ExampleInsert() {
	type Model struct {
		// pk indicates primary key column which will be ignored during insert
		ID      int        `sqlb:"col:id;pk;returning"`
		Created *time.Time `sqlb:"col:created_at"`
		Updated *time.Time `sqlb:"col:updated_at;conflict_set:NOW()"`
		Deleted *time.Time `sqlb:"col:deleted_at;conflict_set:NULL"`
	}

	type User struct {
		Model `sqlb:"table:users"`
		// conflict_on indicates the column(s) to check for conflict
		Email string `sqlb:"col:email;conflict_on"`
		// conflict_set without value means to use excluded column value
		Name string `sqlb:"col:name;conflict_set"`
		// conflict_set can accept SQL expressions
		Notes string `sqlb:"col:notes;conflict_set:CASE WHEN users.notes = '' THEN excluded.notes ELSE users.notes END"`
	}

	data := &User{Email: "example@example.com"}
	b := sqlb.NewInsertBuilder()
	b.Debug() // enable debug to see the built query
	defer func() {
		if err := recover(); err != nil {
			// ignore error since db is nil
		}
	}()
	err := mapper.InsertOne(nil, b, data)
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// [sqlb] INSERT INTO users (created_at, updated_at, deleted_at, email, name, notes) VALUES (NULL, NULL, NULL, 'example@example.com', '', '') ON CONFLICT (email) DO UPDATE SET updated_at = NOW(), deleted_at = NULL, name = EXCLUDED.name, notes = CASE WHEN users.notes = '' THEN excluded.notes ELSE users.notes END RETURNING id
}

func ExampleUpdate() {
	type Model struct {
		// pk indicates primary key column which will be used in WHERE clause
		ID int `sqlb:"col:id;pk"`
		// extra match column for WHERE clause
		UserID int `sqlb:"col:user_id;match"`
	}

	type User struct {
		Model `sqlb:"table:users"`
		Email string `sqlb:"col:email"`
		Name  string `sqlb:"col:name"`
	}

	data := &User{
		Model: Model{
			ID:     1,
			UserID: 2,
		},
		Email: "example@example.com",
	}
	b := sqlb.NewUpdateBuilder()
	b.Debug() // enable debug to see the built query
	defer func() {
		if err := recover(); err != nil {
			// ignore error since db is nil
		}
	}()
	err := mapper.Update(nil, b, data)
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// [sqlb] UPDATE users SET email = 'example@example.com', name = '' WHERE id = 1 AND user_id = 2
}
