package sqlb_test

import (
	"database/sql"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/scanner"
	"github.com/qjebbs/go-sqlf/v4"
)

func Example_wrapping() {
	var db *sql.DB
	q := NewUserQueryBuilder(db).
		WithIDs([]int64{1, 2, 3})
	q.GetUsers()
}

// Wrap with your own build to provide more friendly APIs.
type UserQueryBuilder struct {
	scanner.QueryAble
	*sqlb.QueryBuilder
}

var Users = sqlb.NewTable("users", "u")

func NewUserQueryBuilder(db scanner.QueryAble) *UserQueryBuilder {
	b := sqlb.NewQueryBuilder().
		Distinct().
		From(Users)
	//  .InnerJoin(...).
	// 	LeftJoin(...).
	// 	LeftJoinOptional(...)
	return &UserQueryBuilder{db, b}
}

func (b *UserQueryBuilder) WithIDs(ids []int64) *UserQueryBuilder {
	b.WhereIn(Users.Column("id"), ids)
	return b
}

func (b *UserQueryBuilder) GetUsers() ([]*User, error) {
	b.Select(Users.Columns("id", "name", "email")...)
	return scanner.SelectManual(b.QueryAble, b.QueryBuilder, sqlf.BindStyleDollar, func() (*User, []any) {
		r := &User{}
		return r, []interface{}{
			&r.ID, &r.Name, &r.Email,
		}
	})
}

type User struct {
	ID    int64
	Name  string
	Email string
}
