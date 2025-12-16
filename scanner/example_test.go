package scanner_test

import (
	"fmt"
	"time"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/scanner"
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
		// Anonymous field supports only 'default_tables' key.
		// The 'default_tables' declare default tables for its subfields and
		// subsequent sibling fields who don't declare tables explicitly.
		Model `sqlb:"default_tables:u"`
		Name  string `sqlb:"col:name"`
		// Included only when "full" tag is specified
		Notes string `sqlb:"col:notes;on:full"`
	}

	type UserOrg struct {
		// Dive into structs
		User *User `sqlb:"dive"`
		// Unlike col tags, a sel tag semantically suggest that
		// it's a SELECT expression rather than a column.
		OrgName float64 `sqlb:"sel:COALESCE(?.name,'');tables:o"`
	}

	Users := sqlb.NewTable("users", "u")
	Orgs := sqlb.NewTable("orgs", "o")
	b := sqlb.NewQueryBuilder().
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
	_, err := scanner.Select[*UserOrg](nil, b)
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// [sqlb] SELECT u.id, u.created_at, u.updated_at, u.deleted_at, u.name, COALESCE(o.name,'') FROM users AS u LEFT JOIN orgs AS o ON u.org_id = o.id WHERE u.id = 1
}
