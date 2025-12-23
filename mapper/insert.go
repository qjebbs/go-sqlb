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
// Insert omits zero-value fields by default. To include zero-value fields in the INSERT statement, use the `insert_zero` struct tag.
//
// Limitations:
//   - Full tags support for PostgreSQL and SQLite
//   - No 'returning' support for MySQL
//   - No 'conflict_on' / 'conflict_set' support for SQL Server or Oracle
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"pk;col:id;table:users;"`
//
// The supported struct tags are:
//   - table: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields.
//   - col: Specify the column associated with the field.
//   - readonly: The field is excluded from INSERT statement.
//   - insert_zero: Don't omit the field from INSERT statement even if it has zero value.
//   - returning: Mark the field to be included in RETURNING clause.
//   - conflict_on: Declare current as one of conflict detection column.
//   - conflict_set: Update the field on conflict. It's equivalent to `SET <column>=EXCLUDED.<column>` in ON CONFLICT clause if not specified with value, and can be specified with expression, e.g. `conflict_set:NULL`, which is equivalent to `SET <column>=NULL`.
func Insert[T any](db QueryAble, values []T, options ...Option) error {
	if len(values) == 0 {
		return nil
	}
	opt := mergeOptions(options...)
	if opt.dialect == dialects.DialectOracle {
		// Oracle does not support DEFAULT keyword in INSERT VALUES,
		// so we have to insert one by one.
		return wrapErrWithDebugName("Insert", values[0], insertOneByOne(db, values, opt))
	}
	return wrapErrWithDebugName("Insert", values[0], insert(db, values, opt))
}

func insertOneByOne[T any](db QueryAble, values []T, opt *Options) error {
	for _, v := range values {
		if err := insert(db, []T{v}, opt); err != nil {
			return err
		}
	}
	return nil
}

func insert[T any](db QueryAble, values []T, opt *Options) error {
	if err := checkPtrStruct(values[0]); err != nil {
		return err
	}
	queryStr, args, returningFields, err := buildInsertQueryForStruct(values, opt)
	if err != nil {
		return err
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
	insertInfo, err := buildInsertInfo(opt.dialect, info, values)
	if err != nil {
		return "", nil, nil, err
	}
	b := sqlb.NewInsertBuilder(opt.dialect).
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
	if opt.debug {
		b.Debug(debugName("Insert", values[0]))
	}

	query, args, err = b.BuildQuery(opt.style)
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

func buildInsertInfo[T any](dialect dialects.Dialect, f *structInfo, values []T) (insertInfo, error) {
	var r insertInfo
	reflectValues := util.Map(values, func(v T) reflect.Value {
		rv := reflect.ValueOf(v)
		for rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		return rv
	})
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
		colIndent := dialect.QuoteIdentifier(col.Column)
		allZero := true
		colValues := util.Map(reflectValues, func(v reflect.Value) any {
			field := v.FieldByIndex(col.Index)
			if field.IsZero() {
				if !col.InsertZero {
					return defaultBuilder
				}
			} else if allZero {
				allZero = false
			}
			return field.Interface()
		})
		if col.Returning {
			r.returningColumns = append(r.returningColumns, colIndent)
			r.returningFields = append(r.returningFields, col)
		}
		if col.ConflictOn {
			switch dialect {
			case dialects.DialectPostgreSQL, dialects.DialectSQLite:
				r.conflict = append(r.conflict, colIndent)
			case dialects.DialectMySQL:
				// ignore, MySQL uses different syntax
			default:
				return r, fmt.Errorf("does not support 'conflict_on' tag for %s", dialect)
			}
		}
		if col.ConflictSet != nil {
			colQuoted := sqlf.F(colIndent)
			var setValue sqlf.Builder
			if *col.ConflictSet == "" {
				switch dialect {
				case dialects.DialectPostgreSQL, dialects.DialectSQLite:
					setValue = sqlf.F("EXCLUDED.?", colQuoted)
				case dialects.DialectMySQL:
					setValue = sqlf.F("VALUES(?)", colQuoted)
				case dialects.DialectSQLServer, dialects.DialectOracle:
					return r, fmt.Errorf("'conflict_set' is not supported for %s", dialect)
				default:
					return r, fmt.Errorf("'conflict_set' without expression is not supported for %s", dialect)
				}
			} else {
				// user specified expression
				switch dialect {
				case dialects.DialectPostgreSQL, dialects.DialectSQLite, dialects.DialectMySQL:
					setValue = sqlf.F(*col.ConflictSet)
				case dialects.DialectSQLServer, dialects.DialectOracle:
					return r, fmt.Errorf("'conflict_set' is not supported for %s", dialect)
				default:
					return r, fmt.Errorf("'conflict_set' without expression is not supported for %s", dialect)
				}
			}
			r.actions = append(r.actions, sqlf.F("? = ?", colQuoted, setValue))
		}

		if col.PK || col.ReadOnly || !col.InsertZero && allZero {
			continue
		}
		r.insertColumns = append(r.insertColumns, colIndent)
		r.insertIndices = append(r.insertIndices, col.Index)
		r.insertValues = append(r.insertValues, colValues)
	}
	return r, nil
}
