package sqlb

import (
	"fmt"

	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

type depTablesKey struct{}

// NoDeps returns a new builder that doesn't report any dependencies.
//
// *QueryBuilder collects table dependencies so that they can be used
// for JOIN elimination. But it just simply collects all appearances
// of tables in the query, even for those in a self-contained subqueries.
// Wrap subqueries with NoDeps to ignore their dependencies.
//
// For example, the table 'bar' will not count as a dependency of the main query.
//
//	b.Where(sqlb.NoDeps(sqlf.F(
//	  "id IN (SELECT ? FROM ?)",
//	  bar, bar.Column("id"),
//	)))
//
// No need to wrap *QueryBuilder with NoDeps, since it never report any
// dependencies to outer queries.
func NoDeps(b sqlf.Builder) sqlf.Builder {
	return sqlf.Func(func(ctx *sqlf.Context) (query string, err error) {
		if ctx.Value(depTablesKey{}) != nil {
			// the call is to collect dependencies,
			// do nothing since there are no dependencies here.
			return "", nil
		}
		return b.Build(ctx)
	})
}

// collectDependencies collects the dependencies of the tables.
func (b *QueryBuilder) collectDependencies() (map[Table]bool, error) {
	builders := util.Concat(
		b.selects,
		b.touches,
		b.conditions,
		b.groupbys,
		b.havings,
	)
	for _, order := range b.orders {
		builders = append(builders, order.column)
	}

	tables, err := extractTables(builders...)
	if err != nil {
		return nil, fmt.Errorf("collect dependencies: %w", err)
	}
	deps := make(map[Table]bool)
	// first table is the main table and always included
	deps[b.tables[0].table] = true
	for table := range tables {
		err := b.collectDepsFromTable(deps, table)
		if err != nil {
			return nil, err
		}
	}
	// mark for CTEs
	depsCTE := make(map[Table]bool)
	for _, t := range b.tables {
		if (b.distinct || len(b.groupbys) > 0) && t.optional && !deps[t.table] {
			continue
		}
		if cte, ok := b.ctesDict[t.table.AppliedName()]; ok {
			b.collectDepsFromCTE(depsCTE, cte)
		}
	}
	for cte := range depsCTE {
		deps[cte] = true
	}
	return deps, nil
}

func (b *QueryBuilder) collectDepsFromCTE(deps map[Table]bool, cte *cte) error {
	key := cte.table
	if deps[key] {
		return nil
	}
	deps[key] = true
	tables, err := extractTables(cte.Builder)
	if err != nil {
		return fmt.Errorf("collect dependencies of CTE %q: %w", cte.table, err)
	}
	for dep := range tables {
		if cte, ok := b.ctesDict[dep]; ok {
			err := b.collectDepsFromCTE(deps, cte)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *QueryBuilder) collectDepsFromTable(dep map[Table]bool, t string) error {
	from, ok := b.tablesDict[t]
	if !ok {
		return fmt.Errorf("from undefined: '%s'", t)
	}
	if dep[from.table] {
		return nil
	}
	dep[from.table] = true
	tables, err := extractTables(from)
	if err != nil {
		return fmt.Errorf("collect dependencies of table %q: %w", from.table.Name, err)
	}
	for ft := range tables {
		if ft == t {
			continue
		}
		err := b.collectDepsFromTable(dep, ft)
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTables(args ...sqlf.Builder) (map[string]bool, error) {
	tables := make(map[string]bool)
	ctx := sqlf.ContextWith(sqlf.NewContext(sqlf.BindStyleDollar), depTablesKey{}, tables)
	_, err := sqlf.Join(";", args...).Build(ctx)
	if err != nil {
		return nil, err
	}
	return tables, nil
}
