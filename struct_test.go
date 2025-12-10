package sqlb

import (
	"testing"

	"github.com/qjebbs/go-sqlf/v4"
)

func TestQueryStruct(t *testing.T) {
	type Model struct {
		ID int `sqlb:"?.id"`
	}

	type User struct {
		*Model   `sqlb:"u"`
		Name     string `sqlb:"u.name"`
		Email    string `sqlb:"u.email"`
		Constant string `sqlb:"'str'"`

		Child *User

		unexported string `sqlb:"1"` // should be ignored
	}

	userTable := NewTable("users", "u")
	b := NewQueryBuilder().
		From(userTable).Where(sqlf.F(
		"?=?",
		userTable.Column("id"), 1,
	))
	want := "SELECT u.id, u.name, u.email, 'str' FROM users AS u WHERE u.id=$1"
	got, _, err := _BuildQueryForStruct[User](b, sqlf.BindStyleDollar)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
