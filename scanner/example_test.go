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
		ID      int        `sqlb:"col:?.id"`
		Created *time.Time `sqlb:"col:?.created_at"`
		Updated *time.Time `sqlb:"col:?.updated_at"`
		Deleted *time.Time `sqlb:"col:?.deleted_at"`
	}

	type Bill struct {
		// Anonymous field supports only 'default_tables' key.
		// The 'default_tables' declare default tables for its subfields and
		// subsequent sibling fields who don't declare tables explicitly.
		Model  `sqlb:"default_tables:b"`
		Amount float64 `sqlb:"col:?.amount"`
		// Included only when "full" tag is specified
		Notes string `sqlb:"col:?.notes;on:full"`
	}

	type UserBill struct {
		// It's also possible to use a dummy field to
		// define the default table for subsequent sibling fields.
		_     struct{} `sqlb:"default_tables:u"`
		ID    int      `sqlb:"col:?.id"`
		Owner string   `sqlb:"col:?.name"`

		// Dive into the Bill struct
		Bill *Bill `sqlb:"dive;default_tables:b"`
	}

	Users := sqlb.NewTable("users", "u")
	Bills := sqlb.NewTable("bills", "b")
	b := sqlb.NewQueryBuilder().From(Users).LeftJoin(
		Bills, sqlf.F("?.id = ?.user_id", Users, Bills),
	).Limit(10)
	b.Debug() // enable debug to see the built query
	defer func() {
		if err := recover(); err != nil {
			// ignore error since db is nil
		}
	}()
	_, err := scanner.Select[*UserBill](nil, b, scanner.WithTags("full"))
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// [sqlb] SELECT u.id, u.name, b.id, b.created_at, b.updated_at, b.deleted_at, b.amount, b.notes FROM users AS u LEFT JOIN bills AS b ON u.id = b.user_id LIMIT 10
}
