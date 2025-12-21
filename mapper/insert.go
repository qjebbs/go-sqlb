package mapper

import (
	"fmt"
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/dialects"
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// InsertOne inserts a single struct into the database.
// It scans the returning columns into the corresponding fields of value.
// If no returning columns are specified, it only executes the insert query.
//
// See Insert() for supported struct tags.
func InsertOne[T any](db QueryAble, value T, options ...Option) error {
	return Insert(db, []T{value}, options...)
}

// Insert inserts multiple structs into the database.
// It scans the returning columns into the corresponding fields of values.
// If no returning columns are specified, it only executes the insert query.
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"pk;col:id;table:users;"`
//
// The supported struct tags are:
//   - table: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields.
//   - col: Specify the column associated with the field.
//   - returning: Mark the field to be included in RETURNING clause.
//   - conflict_on: Declare current as one of conflict detection column.
//   - conflict_set: Update the field on conflict. It's equivalent to `SET <column>=EXCLUDED.<column>` in ON CONFLICT clause if not specified with value, and can be specified with expression, e.g. `conflict_set:NULL`, which is equivalent to `SET <column>=NULL`.
func Insert[T any](db QueryAble, values []T, options ...Option) error {
	if len(values) == 0 {
		return nil
	}
	if err := checkStruct(values[0]); err != nil {
		return err
	}
	opt := mergeOptions(options...)
	queryStr, args, returningFields, err := buildInsertQueryForStruct(values, opt)
	if err != nil {
		return err
	}
	if len(returningFields) == 0 {
		_, err = db.Exec(queryStr, args...)
		return err
	}
	index := 0
	agents := make([]*nullZeroAgent, 0)
	_, err = scan(db, queryStr, args, func() (T, []any) {
		dest := values[index]
		index++
		dest, fields, ag := prepareScanDestinations(dest, returningFields, opt)
		agents = append(agents, ag...)
		return dest, fields
	})
	if err != nil {
		return err
	}
	for _, agent := range agents {
		agent.Apply()
	}
	return nil
}

func buildInsertQueryForStruct[T any](values []T, opt *Options) (query string, args []any, returningFields []fieldInfo, err error) {
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
	insertInfo := buildInsertInfo(opt.dialect, info)
	b := sqlb.NewInsertBuilder().
		InsertInto(insertInfo.table).
		Columns(insertInfo.insertColumns...)

	for _, row := range util.Map(values, func(v T) []any {
		return collectInsertValues(v, insertInfo)
	}) {
		b.Values(row...)
	}

	if len(insertInfo.conflict) > 0 {
		// allow manual setting of on conflict only if no config in tags
		b.OnConflict(insertInfo.conflict, insertInfo.actions...)
	}
	// no allow manual setting of returning columns, since we cannot map them back
	b.Returning(insertInfo.returningColumns...)
	if opt.debug {
		b.Debug(fmt.Sprintf("Insert(%T)", values[0]))
	}

	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, nil, err
	}
	return query, args, insertInfo.returningFields, nil
}

func collectInsertValues[T any](values T, insertInfo insertInfo) []any {
	valueVal := reflect.ValueOf(values)
	if valueVal.Kind() == reflect.Ptr {
		valueVal = valueVal.Elem()
	}
	var row []any
	for _, indexPath := range insertInfo.insertIndices {
		field := valueVal.FieldByIndex(indexPath)
		row = append(row, field.Interface())
	}
	return row
}

type insertInfo struct {
	table string

	insertColumns []string
	insertIndices [][]int
	conflict      []string
	actions       []sqlf.Builder

	returningColumns []string
	returningFields  []fieldInfo
}

func buildInsertInfo(dialect dialects.Dialect, f *structInfo) insertInfo {
	var r insertInfo
	for _, col := range f.columns {
		if r.table == "" && !col.Diving && col.Table != "" {
			// respect first non-diving column with table specified
			r.table = col.Table
		}
		if col.Column == "" {
			continue
		}
		colIndent := dialect.QuoteIdentifier(col.Column)
		if col.Returning {
			r.returningColumns = append(r.returningColumns, colIndent)
			r.returningFields = append(r.returningFields, col)
		}
		if col.PK {
			continue
		}
		r.insertColumns = append(r.insertColumns, colIndent)
		r.insertIndices = append(r.insertIndices, col.Index)
		if col.ConflictOn {
			r.conflict = append(r.conflict, colIndent)
		}
		if col.ConflictSet != nil {
			colQuoted := sqlf.F(colIndent)
			var setValue sqlf.Builder
			if *col.ConflictSet == "" {
				setValue = sqlf.F("EXCLUDED.?", colQuoted)
			} else {
				setValue = sqlf.F(*col.ConflictSet)
			}
			r.actions = append(r.actions, sqlf.F("? = ?", colQuoted, setValue))
		}
	}
	return r
}
