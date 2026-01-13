package sqlb

import (
	"errors"
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
)

// clauseFrom represents a SQL FROM clause.
type clauseFrom struct {
	tables     []*fromTable          // the tables in order
	tablesDict map[string]*fromTable // the from tables by alias

	explicitFrom bool    // whether From() has been called
	errors       []error // errors during building
}

// newFrom creates a new From instance.
func newFrom() *clauseFrom {
	return &clauseFrom{
		tablesDict: make(map[string]*fromTable),
	}
}

// fromBuilderMeta contains metadata for building FROM clause.
type fromBuilderMeta struct {
	DebugName  string
	DependOnMe []sqlf.Builder
	Distinct   bool
	HasGroupBy bool
}

// BuildRequired builds the FROM clause with required tables.
func (b *clauseFrom) BuildRequired(ctx *sqlf.Context, meta *fromBuilderMeta, deps *dependencies) (string, error) {
	err := b.anyError()
	if err != nil {
		return "", err
	}
	if len(b.tables) == 0 {
		return "", nil
	}
	pruning := pruningFromContext(ctx)
	tables := make([]string, 0, len(b.tables))
	if b.explicitFrom {
		c, err := b.tables[0].BuildTo(ctx)
		if err != nil {
			return "", fmt.Errorf("build FROM '%s': %w", b.tables[0].table, err)
		}
		tables = append(tables, "FROM "+c)
	}
	for _, t := range b.tables[1:] {
		if pruning && b.shouldEliminateTable(meta, t, deps) {
			continue
		}
		c, err := t.Builder.BuildTo(ctx)
		if err != nil {
			return "", fmt.Errorf("build FROM '%s': %w", t.table, err)
		}
		tables = append(tables, c)
	}
	if len(tables) == 0 {
		return "", fmt.Errorf("no FROM tables available after elimination")
	}
	return strings.Join(tables, " "), nil

}

// CollectDependencies collects the table dependencies from FROM clause.
func (b *clauseFrom) CollectDependencies(ctx *sqlf.Context, meta *fromBuilderMeta) (*dependencies, error) {
	// extractTables gets all deps used in the builders,
	// there are two types of table reporting:
	// 1. *SelectBuilder only reports its unresolved deps (not defined in CTEs).
	// 2. sqlf.Table in any other sqlf.Builder always reports itself.
	deps, err := b.extractTables(ctx, meta.DebugName, meta.DependOnMe...)
	if err != nil {
		return nil, fmt.Errorf("collect dependencies: %w", err)
	}
	// outer tables of subqueries is my tables
	for t := range deps.OuterTables {
		deps.Tables[t] = true
	}
	deps.OuterTables = map[Table]bool{}
	depsOfTables := newDependencies()
	for _, t := range b.tables {
		if b.shouldEliminateTable(meta, t, deps) {
			continue
		}
		// required by FROM / JOIN
		deps.Tables[t.table] = true
		// collect deps from FROM / JOIN ON clauses.
		err := b.collectDepsFromTable(ctx, meta, depsOfTables, t.table)
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

func (b *clauseFrom) collectDepsFromTable(ctx *sqlf.Context, meta *fromBuilderMeta, dep *dependencies, t Table) error {
	from, ok := b.tablesDict[t.AppliedName()]
	if !ok {
		if meta.DebugName != "" {
			return fmt.Errorf("[%s] from undefined: '%s'", meta.DebugName, t)
		}
		return fmt.Errorf("from undefined: '%s'", t)
	}
	if dep.Tables[t] {
		return nil
	}
	dep.Tables[t] = true
	tables, err := b.extractTables(ctx, meta.DebugName, from)
	if err != nil {
		return fmt.Errorf("collect dependencies of table %q: %w", from.table.Name, err)
	}
	for ft := range tables.Tables {
		if ft == t {
			continue
		}
		err := b.collectDepsFromTable(ctx, meta, dep, ft)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *clauseFrom) extractTables(ctx *sqlf.Context, debugName string, args ...sqlf.Builder) (*dependencies, error) {
	tables := newDependencies(debugName)
	depCtx := contextWithDependencies(ctx, tables)
	_, err := sqlf.Join(";", args...).BuildTo(depCtx)
	if err != nil {
		return nil, err
	}
	return tables, nil
}
func (b *clauseFrom) shouldEliminateTable(meta *fromBuilderMeta, t *fromTable, dep *dependencies) bool {
	if !t.optional || dep == nil || dep.Tables == nil || dep.Tables[t.table] {
		return false
	}
	// automatic elimination for LEFT JOIN tables
	if meta.Distinct || meta.HasGroupBy {
		return true
	}
	return t.forceEliminate
}

// From set the from table.
func (b *clauseFrom) From(t Table) *clauseFrom {
	b.explicitFrom = true
	return b.from(t)
}

// ImplicitedFrom set the from table only for dependency collection,
// but ignore it in the final query building (for UPDATE .. JOIN ..).
// It has no effect if From() has been called before.
func (b *clauseFrom) ImplicitedFrom(t Table) *clauseFrom {
	b.explicitFrom = false
	return b.from(t)
}

func (b *clauseFrom) from(t Table) *clauseFrom {
	if t.Name == "" {
		b.pushError(fmt.Errorf("from table is empty"))
		return b
	}
	table := &fromTable{
		table:          t,
		Builder:        t.TableAs(),
		optional:       false,
		forceEliminate: false,
	}
	if len(b.tables) == 0 {
		b.tables = append(b.tables, table)
	} else {
		b.tables[0] = table
	}
	b.tablesDict[t.AppliedName()] = table
	return b
}

// Join append or replace a Join table.
func (b *clauseFrom) Join(joinStr string, t Table, on *sqlf.Fragment, optional, forceEliminate bool) *clauseFrom {
	if t.Name == "" {
		b.pushError(fmt.Errorf("join table name is empty"))
		return b
	}
	// if _, ok := b.tablesDict[t.AppliedName()]; ok {
	// 	if t.Alias == "" {
	// 		b.pushError(fmt.Errorf("table [%s] is already joined", t.Name))
	// 		return b
	// 	}
	// 	b.pushError(fmt.Errorf("table [%s AS %s] is already joined", t.Name, t.Alias))
	// 	return b
	// }
	if len(b.tables) == 0 {
		// reserve the first alias for the main table
		b.tables = append(b.tables, &fromTable{})
	}
	table := &fromTable{
		table: t,
		Builder: sqlf.F(
			joinStr+" ? ?",
			t.TableAs(),
			sqlf.Prefix("ON", on),
		),
		optional:       optional,
		forceEliminate: optional && forceEliminate,
	}
	if target, replacing := b.tablesDict[t.AppliedName()]; replacing {
		*target = *table
		return b
	}
	b.tables = append(b.tables, table)
	b.tablesDict[t.AppliedName()] = table
	return b
}

type fromTable struct {
	sqlf.Builder
	table          Table
	optional       bool // only for auto-elimination of LEFT JOIN
	forceEliminate bool // user declared to eliminate if not referenced
}

func (b *clauseFrom) pushError(err error) {
	b.errors = append(b.errors, err)
}

func (b *clauseFrom) anyError() error {
	if len(b.errors) == 0 {
		return nil
	}
	sb := new(strings.Builder)
	sb.WriteString("collected errors: \n")
	for _, err := range b.errors {
		sb.WriteString(" - ")
		sb.WriteString(err.Error())
		sb.WriteRune('\n')
	}
	return errors.New(sb.String())
}
