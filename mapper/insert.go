package mapper

import (
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ InsertBuilder = (*sqlb.InsertBuilder)(nil)

// InsertBuilder is the interface for builders that support Insert method.
type InsertBuilder interface {
	sqlb.Builder
	SetInsertTable(table string)
	SetColumns(columns []string)
	SetValues(rows [][]any)
	SetOnConflict(columns []string, actions []sqlf.Builder)
	SetReturning(columns []string)
}

// InsertOne inserts a single struct into the database.
func InsertOne[T any](db QueryAble, b InsertBuilder, value T, options ...Option) error {
	return Insert(db, b, []T{value}, options...)
}

// Insert executes the query and scans the results into a slice of struct T.
// If no returning columns are specified, it only executes the insert query.
// Otherwise, it scans the returning columns into the corresponding fields of T.
func Insert[T any](db QueryAble, b InsertBuilder, values []T, options ...Option) error {
	if len(values) == 0 {
		return nil
	}
	opt := mergeOptions(options...)
	queryStr, args, returningFieldIndices, err := buildInsertQueryForStruct[T](b, values, opt)
	if err != nil {
		return err
	}
	if len(returningFieldIndices) == 0 {
		_, err = db.Exec(queryStr, args...)
		return err
	}
	index := 0
	_, err = scan(db, queryStr, args, func() (T, []any) {
		dest := values[index]
		index++
		return prepareScanDestinations(dest, returningFieldIndices)
	})
	return err
}

func buildInsertQueryForStruct[T any](b InsertBuilder, values []T, opt *Options) (query string, args []any, fieldIndices [][]int, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	var zero T
	info, err := getStructInfo(zero)
	if err != nil {
		return "", nil, nil, err
	}
	insertInfo := buildInsertInfo(opt.dialect, info)
	if insertInfo.table != "" {
		// don't override with empty table in case the table is set manually
		b.SetInsertTable(insertInfo.table)
	}
	b.SetColumns(insertInfo.insertColumns)
	b.SetValues(util.Map(values, func(v T) []any {
		return collectInsertValues(v, insertInfo)
	}))
	if len(insertInfo.conflict) > 0 {
		// allow manual setting of on conflict only if no config in tags
		b.SetOnConflict(insertInfo.conflict, insertInfo.actions)
	}
	// no allow manual setting of returning columns, since we cannot map them back
	b.SetReturning(insertInfo.returningColumns)
	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, nil, err
	}
	return query, args, insertInfo.returningIndices, nil
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
	returningIndices [][]int
}

func buildInsertInfo(dialect Dialect, f *structInfo) insertInfo {
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
			r.returningIndices = append(r.returningIndices, col.Index)
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
