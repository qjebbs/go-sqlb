package sqlb

import (
	"fmt"

	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

type selectBuilderDependencies struct {
	queryDeps  *depTables
	cteDeps    map[string]bool
	unresolved *depTables
}

// collectDependencies collects the dependencies of the tables.
func (b *SelectBuilder) collectDependencies() (*selectBuilderDependencies, error) {
	if b.deps != nil {
		return b.deps, nil
	}
	queryDeps, err := b.collectQueryDependencies()
	if err != nil {
		return nil, err
	}
	cteRequired, cteUnresolved, err := b.ctes.collectDependenciesForTables(queryDeps)
	if err != nil {
		return nil, err
	}
	r := &selectBuilderDependencies{
		queryDeps:  queryDeps,
		cteDeps:    cteRequired,
		unresolved: cteUnresolved,
	}
	b.deps = r
	// if b.debug && b.debugName != "" {
	// 	fmt.Printf("[%s] unresolved: %s\n", b.debugName, util.Map(
	// 		util.MapKeys(r.unresolved),
	// 		func(t Table) string { return t.Name },
	// 	))
	// }
	return r, nil
}

func (b *SelectBuilder) collectQueryDependencies() (*depTables, error) {
	builders := util.Concat(
		b.selects,
		b.conditions,
		util.Map(b.orders, func(i *orderItem) sqlf.Builder { return i.column }),
		b.groupbys,
		b.havings,
		b.unions,
	)
	for _, order := range b.orders {
		builders = append(builders, order.column)
	}

	// extractTables gets all deps used in the builders,
	// there are two types of table reporting:
	// 1. *SelectBuilder only reports its unresolved deps (not defined in CTEs).
	// 2. sqlf.Table in any other sqlf.Builder always reports itself.
	deps, err := b.extractTables(builders...)
	if err != nil {
		return nil, fmt.Errorf("collect dependencies: %w", err)
	}
	// outer tables of subqueries is my tables
	for t := range deps.OuterTables {
		deps.Tables[t] = true
	}
	deps.OuterTables = map[Table]bool{}
	depsOfTables := newDepTables()
	for _, t := range b.tables {
		if b.shouldEliminateTable(t, deps) {
			continue
		}
		// required by FROM / JOIN
		deps.Tables[t.table] = true
		// collect deps from FROM / JOIN ON clauses.
		err := b.collectDepsFromTable(depsOfTables, t.table)
		if err != nil {
			return nil, err
		}
	}
	deps.Merge(depsOfTables)
	for name := range deps.Tables {
		// only respect the applied name of 'name', since it's
		// unique and always valid in SelectBuilder
		if t, ok := b.tablesDict[name.AppliedName()]; ok {
			// required by FROM / JOIN
			deps.SourceNames[t.table.Name] = true
			if t.table != name {
				// t.Name may be empty (from sqlb tag),
				// or even wrong across builder scopes.
				delete(deps.Tables, name)
				deps.Tables[t.table] = true
			}
		} else {
			// require outer FROM / JOIN
			delete(deps.Tables, name)
			deps.OuterTables[name] = true
		}
	}

	return deps, nil
}

func (b *SelectBuilder) collectDepsFromTable(dep *depTables, t Table) error {
	from, ok := b.tablesDict[t.AppliedName()]
	if !ok {
		if b.debugName != "" {
			return fmt.Errorf("[%s] from undefined: '%s'", b.debugName, t)
		}
		return fmt.Errorf("from undefined: '%s'", t)
	}
	if dep.Tables[t] {
		return nil
	}
	dep.Tables[t] = true
	tables, err := b.extractTables(from)
	if err != nil {
		return fmt.Errorf("collect dependencies of table %q: %w", from.table.Name, err)
	}
	for ft := range tables.Tables {
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

func (b *SelectBuilder) extractTables(args ...sqlf.Builder) (*depTables, error) {
	tables := newDepTables(b.debugName)
	ctx := contextWithDepTables(sqlf.NewContext(sqlf.BindStyleDollar), tables)
	_, err := sqlf.Join(";", args...).Build(ctx)
	if err != nil {
		return nil, err
	}
	return tables, nil
}
