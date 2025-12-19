package clauses

import (
	"errors"
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
)

// From represents a SQL FROM clause.
type From struct {
	tables     []*fromTable          // the tables in order
	tablesDict map[string]*fromTable // the from tables by alias

	explicitFrom bool    // whether From() has been called
	errors       []error // errors during building
}

// NewFrom creates a new From instance.
func NewFrom() *From {
	return &From{
		tablesDict: make(map[string]*fromTable),
	}
}

// FromBuilderMeta contains metadata for building FROM clause.
type FromBuilderMeta struct {
	DebugName  string
	DependOnMe []sqlf.Builder
	Distinct   bool
	HasGroupBy bool
}

// BuildRequired builds the FROM clause with required tables.
func (b *From) BuildRequired(ctx *sqlf.Context, meta *FromBuilderMeta, deps *Dependencies) (string, error) {
	err := b.anyError()
	if err != nil {
		return "", err
	}
	if len(b.tables) == 0 {
		return "", nil
	}
	tables := make([]string, 0, len(b.tables))
	if b.explicitFrom {
		c, err := b.tables[0].Build(ctx)
		if err != nil {
			return "", fmt.Errorf("build FROM '%s': %w", b.tables[0].table, err)
		}
		tables = append(tables, "FROM "+c)
	}
	for _, t := range b.tables[1:] {
		if b.shouldEliminateTable(meta, t, deps) {
			continue
		}
		c, err := t.Builder.Build(ctx)
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
func (b *From) CollectDependencies(meta *FromBuilderMeta) (*Dependencies, error) {
	// extractTables gets all deps used in the builders,
	// there are two types of table reporting:
	// 1. *SelectBuilder only reports its unresolved deps (not defined in CTEs).
	// 2. sqlf.Table in any other sqlf.Builder always reports itself.
	deps, err := b.extractTables(meta.DebugName, meta.DependOnMe...)
	if err != nil {
		return nil, fmt.Errorf("collect dependencies: %w", err)
	}
	// outer tables of subqueries is my tables
	for t := range deps.OuterTables {
		deps.Tables[t] = true
	}
	deps.OuterTables = map[Table]bool{}
	depsOfTables := NewDependencies()
	for _, t := range b.tables {
		if b.shouldEliminateTable(meta, t, deps) {
			continue
		}
		// required by FROM / JOIN
		deps.Tables[t.table] = true
		// collect deps from FROM / JOIN ON clauses.
		err := b.collectDepsFromTable(meta, depsOfTables, t.table)
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

func (b *From) collectDepsFromTable(meta *FromBuilderMeta, dep *Dependencies, t Table) error {
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
	tables, err := b.extractTables(meta.DebugName, from)
	if err != nil {
		return fmt.Errorf("collect dependencies of table %q: %w", from.table.Name, err)
	}
	for ft := range tables.Tables {
		if ft == t {
			continue
		}
		err := b.collectDepsFromTable(meta, dep, ft)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *From) extractTables(debugName string, args ...sqlf.Builder) (*Dependencies, error) {
	tables := NewDependencies(debugName)
	ctx := ContextWithDependencies(sqlf.NewContext(sqlf.BindStyleDollar), tables)
	_, err := sqlf.Join(";", args...).Build(ctx)
	if err != nil {
		return nil, err
	}
	return tables, nil
}
func (b *From) shouldEliminateTable(meta *FromBuilderMeta, t *fromTable, dep *Dependencies) bool {
	if !t.optional || dep.Tables[t.table] {
		return false
	}
	// automatic elimination for LEFT JOIN tables
	if meta.Distinct || meta.HasGroupBy {
		return true
	}
	return t.forceEliminate
}

// From set the from table.
func (b *From) From(t Table) *From {
	b.explicitFrom = true
	return b.from(t)
}

// ImplicitedFrom set the from table only for dependency collection,
// but ignore it in the final query building (for UPDATE .. JOIN ..).
// It has no effect if From() has been called before.
func (b *From) ImplicitedFrom(t Table) *From {
	if b.explicitFrom {
		return b
	}
	return b.from(t)
}

func (b *From) from(t Table) *From {
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
func (b *From) Join(joinStr string, t Table, on *sqlf.Fragment, optional, forceEliminate bool) *From {
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

func (b *From) pushError(err error) {
	b.errors = append(b.errors, err)
}

func (b *From) anyError() error {
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
