package sqlb_test

import (
	"context"
	"database/sql"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlb/mapper"
)

func Example_wrapping() {
	var db *sql.DB
	q := NewUserSelectBuilder(db).
		WithIDs([]int64{1, 2, 3})
	q.GetUsers(context.Background())
}

// Wrap with your own build to provide more friendly APIs.
type UserSelectBuilder struct {
	mapper.QueryAble
	*sqlb.SelectBuilder
}

var Users = sqlb.NewTable("users", "u")

func NewUserSelectBuilder(db mapper.QueryAble) *UserSelectBuilder {
	b := sqlb.NewSelectBuilder().
		Distinct().
		From(Users)
	//  .InnerJoin(...).
	// 	LeftJoin(...).
	// 	LeftJoinOptional(...)
	return &UserSelectBuilder{db, b}
}

func (b *UserSelectBuilder) WithIDs(ids []int64) *UserSelectBuilder {
	b.WhereIn(Users.Column("id"), ids)
	return b
}

func (b *UserSelectBuilder) GetUsers(ctx context.Context) ([]*User, error) {
	b.Select(Users.Columns("id", "name", "email")...)
	buildCtx := sqlb.ContextWithDialect(ctx, dialect.PostgreSQL{})
	return mapper.SelectManual(buildCtx, b.QueryAble, b.SelectBuilder, func() (*User, []any) {
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
