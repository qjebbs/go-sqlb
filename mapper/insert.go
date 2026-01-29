package mapper

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// InsertOne inserts a single struct into the database.
// It scans the returning columns into the corresponding fields of value.
// If no returning columns are specified, it only executes the insert query.
//
// See Insert() for supported struct tags.
func InsertOne[T any](ctx sqlb.Context, db QueryAble, value T, options ...Option) error {
	return Insert(ctx, db, []T{value}, options...)
}

// Insert inserts multiple structs into the database.
// It scans the returning columns into the corresponding fields of values.
// If no returning columns are specified, it only executes the insert query.
//
// Insert omits zero-value fields by default. To include zero-value fields in the INSERT statement, use the `insert_zero` struct tag.
//
// Limitations:
//   - No 'returning' support for MySQL
//   - No 'conflict_on' / 'conflict_set' support for SQL Server or Oracle
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"pk;col:id;table:users;"`
//
// The supported struct tags are:
//   - table<:name>: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields.
//   - col<:name>: Specify the column associated with the field.
//   - pk: The column is primary key, which is excluded from INSERT statement.
//   - unique: The column could be used with conflict_on to detect conflict.
//   - unique_group[:name[,name]...]: The column is one of the "Composite Unique" fields, which could could be used with conflict_on to detect conflict. If there is only one unique_group in the struct, the group name can be omitted.
//   - required: The field is required to have non-zero value, otherwise Insert will return an error.
//   - readonly: The field is excluded from INSERT statement.
//   - insert_zero: Don't omit the field from INSERT statement even if it has zero value.
//   - returning: Mark the field to be included in RETURNING clause.
//   - conflict_on[:unique_group name]: If value is ommited, declare current unique column or current unique-group the conflict detection column(s). If there is any ambiguity, it must be explicitly specified, e.g. `unique_group:a,b,c;conflict_on:a`.
//   - conflict_set: Update the field on conflict. It's equivalent to `SET <column>=EXCLUDED.<column>` in ON CONFLICT clause if not specified with value, and can be specified with expression, e.g. `conflict_set:NULL`, which is equivalent to `SET <column>=NULL`.
func Insert[T any](ctx sqlb.Context, db QueryAble, values []T, options ...Option) error {
	if len(values) == 0 {
		return nil
	}
	opt := mergeOptions(options...)
	if !ctx.Dialect().Capabilities().SupportsInsertDefault {
		// Oracle does not support DEFAULT keyword in INSERT VALUES,
		// so we have to insert one by one.
		return wrapErrWithDebugName("Insert", values[0], insertOneByOne(ctx, db, values, opt))
	}
	return wrapErrWithDebugName("Insert", values[0], insert(ctx, db, values, opt))
}

func insertOneByOne[T any](ctx sqlb.Context, db QueryAble, values []T, opt *Options) error {
	for _, v := range values {
		if err := insert(ctx, db, []T{v}, opt); err != nil {
			return err
		}
	}
	return nil
}

func insert[T any](ctx sqlb.Context, db QueryAble, values []T, opt *Options) error {
	if err := checkPtrStruct(values[0]); err != nil {
		return err
	}
	var debugger *debugger
	if opt.debug {
		debugger = newDebugger("Insert", values[0], opt)
		defer debugger.print(ctx.BaseDialect())
	}
	queryStr, args, returningFields, err := buildInsertQueryForStruct(ctx, values, opt)
	if err != nil {
		return err
	}
	if debugger != nil {
		debugger.onBuilt(queryStr, args)
	}
	if db == nil {
		return ErrNilDB
	}
	if len(returningFields) == 0 {
		_, err = db.Exec(queryStr, args...)
		return err
	}
	index := 0
	agents := make([]*nullZeroAgent, 0)
	_, err = scan(ctx, db, queryStr, args, debugger, func() (T, []any) {
		dest := values[index]
		index++
		dest, fields, ag := prepareScanDestinations(dest, returningFields, opt)
		agents = append(agents, ag...)
		return dest, fields
	})
	if err != nil {
		return err
	}
	if len(agents) > 0 {
		for _, agent := range agents {
			agent.Apply()
		}
		if debugger != nil {
			debugger.onPostScan(nil)
		}
	}
	return nil
}

func buildInsertQueryForStruct[T any](ctx sqlb.Context, values []T, opt *Options) (query string, args []any, returningFields []fieldInfo, err error) {
	if len(values) == 0 {
		return "", nil, nil, fmt.Errorf("no values to insert")
	}
	if opt == nil {
		opt = newDefaultOptions()
	}
	var zero T
	info, err := getStructInfo(zero)
	if err != nil {
		return "", nil, nil, err
	}
	insertInfo, err := buildInsertInfo(ctx.Dialect(), info, values)
	if err != nil {
		return "", nil, nil, err
	}
	b := sqlb.NewInsertBuilder().
		InsertInto(insertInfo.table).
		Columns(insertInfo.insertColumns...)

	for i := 0; i < len(insertInfo.insertValues[0]); i++ {
		var row []any
		for j := 0; j < len(insertInfo.insertColumns); j++ {
			row = append(row, insertInfo.insertValues[j][i])
		}
		b.Values(row...)
	}

	if len(insertInfo.conflict) > 0 || len(insertInfo.actions) > 0 {
		// allow manual setting of on conflict only if no config in tags
		b.OnConflict(insertInfo.conflict, insertInfo.actions...)
	}
	// no allow manual setting of returning columns, since we cannot map them back
	b.Returning(insertInfo.returningColumns...)

	query, args, err = b.Build(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	return query, args, insertInfo.returningFields, nil
}

type insertInfo struct {
	table string

	insertColumns []string
	insertIndices [][]int
	insertValues  [][]any
	conflict      []string
	actions       []sqlf.Builder

	returningColumns []string
	returningFields  []fieldInfo
}

func buildInsertInfo[T any](dialect dialect.Dialect, f *structInfo, values []T) (insertInfo, error) {
	var r insertInfo
	reflectValues := util.Map(values, func(v T) reflect.Value {
		rv := reflect.ValueOf(v)
		for rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		return rv
	})
	var (
		conflictColumns []fieldData
		unique          []fieldData
		uniqueGroups    = make(map[string][]fieldData)
	)

	// FIXME: Oracle does not support DEFAULT keyword in INSERT VALUES
	var defaultBuilder sqlf.Builder = sqlf.F("DEFAULT")
	for _, col := range f.columns {
		if r.table == "" && !col.Diving && col.Table != "" {
			// respect first non-diving column with table specified
			r.table = col.Table
		}
		if col.Column == "" {
			continue
		}
		colIndent := sqlf.Identifier(col.Column)
		fieldData := fieldData{
			Info:          col,
			IndentBuilder: colIndent,
		}
		allZero := true
		noZero := true
		colValues := util.Map(reflectValues, func(v reflect.Value) any {
			field := v.FieldByIndex(col.Index)
			if field.IsZero() {
				noZero = false
				if !col.InsertZero {
					return defaultBuilder
				}
			} else if allZero {
				allZero = false
			}
			return field.Interface()
		})
		if col.Returning {
			r.returningColumns = append(r.returningColumns, col.Column)
			r.returningFields = append(r.returningFields, col)
		}
		if col.Unique {
			unique = append(unique, fieldData)
		}
		if len(col.UniqueGroups) > 0 {
			for _, group := range col.UniqueGroups {
				uniqueGroups[group] = append(uniqueGroups[group], fieldData)
			}
		}
		caps := dialect.Capabilities()
		if col.ConflictOn != nil {
			if !caps.SupportsOnConflict {

				return r, fmt.Errorf("does not support 'conflict_on' tag for dialect %T", dialect)
			}
			conflictColumns = append(conflictColumns, fieldData)
		}
		if col.ConflictSet != nil {
			var setValue sqlf.Builder
			if *col.ConflictSet == "" {
				switch {
				case caps.SupportsOnConflictSetExcluded:
					setValue = sqlf.F("EXCLUDED.?", colIndent)
				case caps.SupportsOnDuplicateKeyUpdate:
					setValue = sqlf.F("VALUES(?)", colIndent)
				default:
					return r, fmt.Errorf("'conflict_set' without expression is not supported for dialect %T", dialect)
				}
			} else {
				// user specified expression
				switch {
				case caps.SupportsOnConflictSetExcluded || caps.SupportsOnDuplicateKeyUpdate:
					setValue = sqlf.F(*col.ConflictSet)
				default:
					return r, fmt.Errorf("'conflict_set' without expression is not supported for dialect %T", dialect)
				}
			}
			r.actions = append(r.actions, sqlf.F("? = ?", colIndent, setValue))
		}

		if !col.InsertZero && col.Info.Required && !noZero {
			if len(values) == 1 {
				return r, fmt.Errorf("%s is required", col.Name)
			}
			return r, fmt.Errorf("%s is required but missing in some of the inserted values", col.Name)
		}
		if col.PK || col.ReadOnly || !col.InsertZero && allZero {
			continue
		}
		r.insertColumns = append(r.insertColumns, col.Column)
		r.insertIndices = append(r.insertIndices, col.Index)
		r.insertValues = append(r.insertValues, colValues)
	}
	// check conflict columns
	if len(conflictColumns) == 0 {
		return r, nil
	}
	conflictFields := make([]string, 0)
	conflictOnGroups := make(map[string]bool)
	for _, conflict := range conflictColumns {
		group, err := getConflictOnGroup(conflict)
		if err != nil {
			return r, err
		}
		if group != "" {
			conflictOnGroups[group] = true
		} else {
			conflictFields = append(conflictFields, conflict.Info.Column)
		}
	}
	nConflictOn := len(conflictFields) + len(conflictOnGroups)
	switch {
	case nConflictOn == 1:
		if len(conflictFields) == 1 {
			r.conflict = conflictFields
		} else {
			// use unique group
			for group := range conflictOnGroups {
				r.conflict = util.Map(uniqueGroups[group], func(f fieldData) string { return f.Info.Column })
			}
		}
	case nConflictOn == 0:
		// should not happen
		return r, fmt.Errorf("has conflict_on fields but no conflict result")
	case nConflictOn > 1:
		if len(conflictFields) == 0 {
			groups := strings.Join(util.MapKeys(conflictOnGroups), ",")
			return r, fmt.Errorf("conflict on multiple unique_group %q", groups)
		}
		if len(conflictOnGroups) == 0 {
			cols := strings.Join(conflictFields, ",")
			return r, fmt.Errorf("conflict on multiple unique %q", cols)
		}
		cols := strings.Join(conflictFields, ",")
		groups := strings.Join(util.MapKeys(conflictOnGroups), ",")
		return r, fmt.Errorf("conflict on both unique %q and unique_group %q", cols, groups)
	}
	return r, nil
}

func getConflictOnGroup(field fieldData) (string, error) {
	if field.Info.ConflictOn == nil {
		// should not happen
		return "", fmt.Errorf("call calcConflictOn() on non-conflict-on column")
	}
	// "conflict_on:group1" conflict on on group1
	if *field.Info.ConflictOn != "" {
		return *field.Info.ConflictOn, nil
	}
	// "unique;conflict_on" conflict on current unique column
	if field.Info.Unique {
		return "", nil
	}
	// "unique_group:group1,group2;conflict_on"
	if len(field.Info.UniqueGroups) > 1 {
		return "", fmt.Errorf("ambiguity 'conflict_on' to 'unique_group:%s', must specify a group name", strings.Join(field.Info.UniqueGroups, ","))
	}
	if len(field.Info.UniqueGroups) == 0 {
		return "", fmt.Errorf("field %q is not unique or unique_group, cannot be used for conflict detection", field.Info.Column)
	}
	// "unique_group:group1;conflict_on" conflict on on group1
	return field.Info.UniqueGroups[0], nil
}
