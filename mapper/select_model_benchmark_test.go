package mapper_test

import (
	"context"
	"testing"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlb/mapper"
	"github.com/qjebbs/go-sqlf/v4"
)

func BenchmarkSelectModelScan(b *testing.B) {
	dest := &userListItem{}
	nFields := len(dest.Values())
	indexes := make([]int, nFields)
	for i := 0; i < nFields; i++ {
		indexes[i] = i
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		values := make([]any, len(indexes))
		dest.FillValues(values, indexes)
	}
}

func BenchmarkSelectModelBuild(b *testing.B) {
	ctx := sqlb.NewContext(context.Background(), dialect.PostgreSQL{})
	for i := 0; i < b.N; i++ {
		_, _ = mapper.Select[*userListItem](ctx, nil, makeBuilder())
	}
}

func BenchmarkSelectReflectBuild(b *testing.B) {
	// not using generated code to test the performance of reflection-based struct parsing and field access in
	type userListItem struct {
		User
		Org
	}
	ctx := sqlb.NewContext(context.Background(), dialect.PostgreSQL{})
	for i := 0; i < b.N; i++ {
		_, _ = mapper.Select[*userListItem](ctx, nil, makeBuilder())
	}
}

func makeBuilder() *sqlb.SelectBuilder {
	user := &User{}
	org := &Org{}
	b := sqlb.NewSelectBuilder()

	b.From(user.Table()).
		InnerJoin(org.Table(), sqlf.F(
			"? = ?",
			user.ColumnOrgID(),
			org.ColumnID(),
		)).
		WhereEquals(org.ColumnID(), 1).
		WhereIsNull(user.ColumnDeleted())
	return b
}
