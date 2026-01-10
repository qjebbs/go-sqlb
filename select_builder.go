package sqlb

import (
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*SelectBuilder)(nil)
var _ Builder = (*SelectBuilder)(nil)

// SelectBuilder is the SQL query sqlb.
// It's recommended to wrap it with your struct to provide a
// more friendly API and improve fragment reusability.
type SelectBuilder struct {
	dialact Dialect

	ctes *clauseWith
	from *clauseFrom

	selects  *clauseList // select columns and keep values in scanning.
	where    *clauseList
	order    *clauseList // order by columns, joined with comma.
	groupbys *clauseList // group by columns, joined with comma.
	having   *clauseList // having conditions, joined with AND.
	distinct bool        // select distinct
	limit    int64       // limit count
	offset   int64       // offset count
	unions   *clauseList // union queries
	errors   []error     // errors during building

	debugger

	pruning bool
	deps    *selectBuilderDependencies
}

// NewSelectBuilder returns a new SelectBuilder.
func NewSelectBuilder(dialect ...Dialect) *SelectBuilder {
	d := DialectPostgres
	if len(dialect) > 0 {
		d = dialect[0]
	}
	return &SelectBuilder{
		dialact:  d,
		ctes:     newWith(),
		from:     newFrom(),
		order:    newPrefixedList("ORDER BY", ", "),
		groupbys: newPrefixedList("GROUP BY", ", "),
		having:   newPrefixedList("HAVING", " AND "),
		selects:  newPrefixedList("SELECT", ", "),
		where:    newPrefixedList("WHERE", " AND "),
		unions:   newPrefixedList("", " "),
	}
}

// Distinct set the flag for SELECT DISTINCT.
func (b *SelectBuilder) Distinct() *SelectBuilder {
	b.distinct = true
	return b
}

// Indistinct unset the flag for SELECT DISTINCT.
func (b *SelectBuilder) Indistinct() *SelectBuilder {
	b.distinct = false
	return b
}

// Select set the columns in the SELECT clause.
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
//
//	foo := sqlb.NewTable("foo")
//	b.Select(foo.Column("bar"))
func (b *SelectBuilder) Select(columns ...sqlf.Builder) *SelectBuilder {
	b.SetSelect(columns...)
	return b
}

// SetSelect set the columns in the SELECT clause,
// which implements the SelectBuilder interface.
func (b *SelectBuilder) SetSelect(columns ...sqlf.Builder) {
	b.resetDepTablesCache()
	b.selects.Replace(columns)
}

// Limit set the limit.
func (b *SelectBuilder) Limit(limit int64) *SelectBuilder {
	b.SetLimit(limit)
	return b
}

// SetLimit implements the SelectLimitBuilder interface.
func (b *SelectBuilder) SetLimit(limit int64) {
	if limit > 0 {
		b.limit = limit
	}
}

// Offset set the offset.
func (b *SelectBuilder) Offset(offset int64) *SelectBuilder {
	if offset > 0 {
		b.offset = offset
	}
	return b
}

// OrderBy set the sorting order.
//
//	foo := sqlb.NewTable("foo")
//	b.OrderBy(sqlf.F("? DESC", foo.Column("bar")))
func (b *SelectBuilder) OrderBy(order ...sqlf.Builder) *SelectBuilder {
	b.resetDepTablesCache()
	b.order.Append(order...)
	return b
}

// GroupBy set the sorting order.
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
//
//	foo := sqlb.NewTable("foo")
//	b.GroupBy(foo.Column("bar"))
func (b *SelectBuilder) GroupBy(columns ...sqlf.Builder) *SelectBuilder {
	b.resetDepTablesCache()
	b.groupbys.Append(columns...)
	return b
}

// Union unions other builders.
//
// !!! Make sure the all table references within the builders are built from sqlb.Table
// to have their dependencies tracked.
func (b *SelectBuilder) Union(builders ...sqlf.Builder) *SelectBuilder {
	b.resetDepTablesCache()
	b.unions.Append(util.Map(builders, func(b sqlf.Builder) sqlf.Builder {
		return sqlf.Prefix("UNION", b)
	})...)
	return b
}

// UnionAll unions other builders with 'UNION ALL'.
//
// !!! Make sure the all table references within the builders are built from sqlb.Table
// to have their dependencies tracked.
func (b *SelectBuilder) UnionAll(builders ...sqlf.Builder) *SelectBuilder {
	b.resetDepTablesCache()
	b.unions.Append(util.Map(builders, func(b sqlf.Builder) sqlf.Builder {
		return sqlf.Prefix("UNION ALL", b)
	})...)
	return b
}

func (b *SelectBuilder) resetDepTablesCache() {
	b.deps = nil
}
