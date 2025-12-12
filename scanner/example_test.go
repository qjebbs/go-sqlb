package scanner_test

import (
	"fmt"
	"time"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/scanner"
)

func ExampleSelect() {
	type Model struct {
		ID      int        `sqlb:"col:?.id"`
		Created *time.Time `sqlb:"col:?.created_at"`
		Updated *time.Time `sqlb:"col:?.updated_at"`
		Deleted *time.Time `sqlb:"col:?.deleted_at"`
	}

	type User struct {
		// Anonymous field supports only 'default_tables' key.
		// The 'default_tables' declare default tables for its subfields and
		// subsequent sibling fields who don't declare tables explicitly.
		*Model `sqlb:"default_tables:u"`
		// If you don't use anonymous field, a dummy field can be used to
		// define the default table for subsequent sibling fields.
		_ struct{} `sqlb:"default_tables:u"`

		Name  string `sqlb:"col:?.name"`            // Inherits table "u" from above
		Age   int    `sqlb:"col:COALESCE(?.age,0)"` // Equals to sqlf.F("COALESCE(?.age,0)", u)
		Notes string `sqlb:"col:?.notes;on:full"`   // Included only when "full" tag is specified
	}

	Users := sqlb.NewTable("users", "u")
	b := sqlb.NewQueryBuilder().From(Users).Limit(10)
	b.Debug() // enable debug to see the built query
	defer func() {
		if err := recover(); err != nil {
			// ignore error since db is nil
		}
	}()
	// u.notes is included because of the "full" tag
	_, err := scanner.Select[*User](nil, b, scanner.WithTags("full"))
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// [sqlb] SELECT u.id, u.created_at, u.updated_at, u.deleted_at, u.name, COALESCE(u.age,0), u.notes FROM users AS u LIMIT 10
}
