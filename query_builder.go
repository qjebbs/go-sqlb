package sqlb

import (
	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/util"
)

// QueryBuilder is the SQL query builder.
// It's recommended to wrap it with your struct to provide a
// more friendly API and improve fragment reusability.
type QueryBuilder struct {
	ctes     []*cte          // common table expressions in order
	ctesDict map[string]*cte // the ctes by name, not alias

	tables     []*fromTable          // the tables in order
	tablesDict map[string]*fromTable // the from tables by alias

	selects    []sqlf.Builder // select columns and keep values in scanning.
	touches    []sqlf.Builder // select columns but drop values in scanning.
	conditions []sqlf.Builder // where conditions, joined with AND.
	orders     []*orderItem   // order by columns, joined with comma.
	groupbys   []sqlf.Builder // group by columns, joined with comma.
	havings    []sqlf.Builder // having conditions, joined with AND.
	distinct   bool           // select distinct
	limit      int64          // limit count
	offset     int64          // offset count
	unions     []sqlf.Builder // union queries

	errors []error // errors during building

	debug     bool // debug mode
	debugName string

	depTablesCache map[Table]bool
}

// NewQueryBuilder returns a new QueryBuilder.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		ctesDict:   make(map[string]*cte),
		tablesDict: make(map[string]*fromTable),
	}
}

// Distinct set the flag for SELECT DISTINCT.
func (b *QueryBuilder) Distinct() *QueryBuilder {
	b.distinct = true
	return b
}

// Indistinct unset the flag for SELECT DISTINCT.
func (b *QueryBuilder) Indistinct() *QueryBuilder {
	b.distinct = false
	return b
}

// SelectReplace replace the columns in the SELECT clause.
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
//
//	foo := sqlb.NewTable("foo")
//	b.SelectReplace(foo.Column("bar"))
func (b *QueryBuilder) SelectReplace(columns ...sqlf.Builder) *QueryBuilder {
	b.resetDepTablesCache()
	b.selects = columns
	return b
}

// Select append the SELECT clause with the columns.
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
//
//	foo := sqlb.NewTable("foo")
//	b.Select(foo.Column("bar"))
func (b *QueryBuilder) Select(columns ...sqlf.Builder) *QueryBuilder {
	if len(columns) == 0 {
		return b
	}
	b.resetDepTablesCache()
	b.selects = append(b.selects, columns...)
	return b
}

// Limit set the limit.
func (b *QueryBuilder) Limit(limit int64) *QueryBuilder {
	if limit > 0 {
		b.limit = limit
	}
	return b
}

// Offset set the offset.
func (b *QueryBuilder) Offset(offset int64) *QueryBuilder {
	if offset > 0 {
		b.offset = offset
	}
	return b
}

// GroupBy set the sorting order.
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
//
//	foo := sqlb.NewTable("foo")
//	b.GroupBy(foo.Column("bar"))
func (b *QueryBuilder) GroupBy(columns ...sqlf.Builder) *QueryBuilder {
	b.resetDepTablesCache()
	b.groupbys = append(b.groupbys, columns...)
	return b
}

// Union unions other builders.
//
// !!! Make sure the all table references within the builders are built from sqlb.Table
// to have their dependencies tracked.
func (b *QueryBuilder) Union(builders ...sqlf.Builder) *QueryBuilder {
	b.resetDepTablesCache()
	b.unions = append(b.unions, util.Map(builders, func(b sqlf.Builder) sqlf.Builder {
		return sqlf.Prefix("UNION", b)
	})...)
	return b
}

// UnionAll unions other builders with 'UNION ALL'.
//
// !!! Make sure the all table references within the builders are built from sqlb.Table
// to have their dependencies tracked.
func (b *QueryBuilder) UnionAll(builders ...sqlf.Builder) *QueryBuilder {
	b.resetDepTablesCache()
	b.unions = append(b.unions, util.Map(builders, func(b sqlf.Builder) sqlf.Builder {
		return sqlf.Prefix("UNION ALL", b)
	})...)
	return b
}

func (b *QueryBuilder) resetDepTablesCache() {
	b.depTablesCache = nil
}
