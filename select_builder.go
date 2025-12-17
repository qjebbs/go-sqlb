package sqlb

import (
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*SelectBuilder)(nil)
var _ Builder = (*SelectBuilder)(nil)

// SelectBuilder is the SQL query builder.
// It's recommended to wrap it with your struct to provide a
// more friendly API and improve fragment reusability.
type SelectBuilder struct {
	ctes       *_CTEs
	tables     []*fromTable          // the tables in order
	tablesDict map[string]*fromTable // the from tables by alias

	selects    []sqlf.Builder // select columns and keep values in scanning.
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

	deps *selectBuilderDependencies
}

// NewSelectBuilder returns a new SelectBuilder.
func NewSelectBuilder() *SelectBuilder {
	return &SelectBuilder{
		ctes:       newCTEs(),
		tablesDict: make(map[string]*fromTable),
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
	b.selects = columns
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

// GroupBy set the sorting order.
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
//
//	foo := sqlb.NewTable("foo")
//	b.GroupBy(foo.Column("bar"))
func (b *SelectBuilder) GroupBy(columns ...sqlf.Builder) *SelectBuilder {
	b.resetDepTablesCache()
	b.groupbys = append(b.groupbys, columns...)
	return b
}

// Union unions other builders.
//
// !!! Make sure the all table references within the builders are built from sqlb.Table
// to have their dependencies tracked.
func (b *SelectBuilder) Union(builders ...sqlf.Builder) *SelectBuilder {
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
func (b *SelectBuilder) UnionAll(builders ...sqlf.Builder) *SelectBuilder {
	b.resetDepTablesCache()
	b.unions = append(b.unions, util.Map(builders, func(b sqlf.Builder) sqlf.Builder {
		return sqlf.Prefix("UNION ALL", b)
	})...)
	return b
}

func (b *SelectBuilder) resetDepTablesCache() {
	b.deps = nil
}
