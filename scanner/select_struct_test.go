package scanner

import (
	"testing"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

func TestQueryStruct(t *testing.T) {
	type Model struct {
		ID int `sqlb:"sel:?.id"`
	}

	type User struct {
		*Model   `sqlb:"default_tables:u"`
		Name     string `sqlb:"col:name"`
		Email    string `sqlb:"col:email"`
		Constant string `sqlb:"sel:'str'"`

		Child *User

		unexported string `sqlb:"sel:1"` // should be ignored
	}

	userTable := sqlb.NewTable("users", "u")
	b := sqlb.NewQueryBuilder().
		From(userTable).Where(sqlf.F(
		"?=?",
		userTable.Column("id"), 1,
	))
	want := "SELECT u.id, u.name, u.email, 'str' FROM users AS u WHERE u.id=$1"
	got, _, _, err := buildQueryForStruct[User](b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
