package sqlb_test

import (
	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

func Example() {
	q := NewUserQueryBuilder(nil).
		WithIDs([]int64{1, 2, 3})
	q.GetUsers()
}

// Wrap with your own build to provide more friendly APIs.
type UserQueryBuilder struct {
	sqlb.QueryAble
	*sqlb.QueryBuilder
}

var Users = sqlb.NewTable("users", "u")

func NewUserQueryBuilder(db sqlb.QueryAble) *UserQueryBuilder {
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
	return sqlb.Query(b.QueryAble, b.QueryBuilder, sqlf.BindStyleDollar, func() (*User, []any) {
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
