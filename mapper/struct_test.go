package mapper

import (
	"context"
	"testing"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlf/v4"
)

func TestQueryStruct(t *testing.T) {
	type Model struct {
		ID int `sqlb:"col:id"`
	}

	type User struct {
		*Model   `sqlb:"table:users"`
		Name     string `sqlb:"col:name"`
		Email    string `sqlb:"col:email"`
		Constant string `sqlb:"sel:'str'"`

		Child *User

		unexported string `sqlb:"sel:1"` // should be ignored
	}

	userTable := sqlb.NewTable("users")
	b := sqlb.NewSelectBuilder().
		From(userTable).Where(sqlf.F(
		"?=?",
		userTable.Column("id"), 1,
	))
	want := `SELECT "users"."id", "users"."name", "users"."email", 'str' FROM "users" WHERE "users"."id"=$1`
	ctx := sqlb.NewContext(context.Background(), dialect.PostgreSQL{})
	got, _, _, err := buildSelectQueryForStruct[User](ctx, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
