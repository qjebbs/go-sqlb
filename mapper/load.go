package mapper

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/dialects"
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// Load loads a struct T from the database.
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"pk;col:id;table:users;"`
//
// The supported struct tags are:
//   - table: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields.
//   - col: The column associated with the field.
//   - load: Custom SQL expression for loading the field, e.g. `COALESCE(?, 0)`, `?` will be replaced with the column identifier.
//   - pk: The column is primary key, which could be used in WHERE clause to locate the row.
//   - unique: The column could be used in WHERE clause to locate the row. There's no tag for "Composite Unique" fields, since any one of them is not unique alone.
//   - conflict_on: Multiple of them form a composite unique constraint, which could be used in WHERE clause to locate the row.
//   - match: The column will be always included in WHERE clause even if it is zero value.
//
// If a struct has all `pk`, `unique`, or `conflict_on` fields zero-valued, the `Load()` operation will return an error.
// If all non-zero-valued, the priority for constructing the WHERE clause is `pk` > `unique` > `conflict_on`.
func Load[T any](db QueryAble, value T, options ...Option) (T, error) {
	r, err := load(db, value, options...)
	if err != nil {
		var zero T
		return zero, wrapErrWithDebugName("Load", zero, err)
	}
	return r, nil
}

func load[T any](db QueryAble, value T, options ...Option) (T, error) {
	var zero T
	if err := checkPtrStruct(value); err != nil {
		return zero, err
	}
	opt := mergeOptions(options...)
	queryStr, args, dests, err := buildLoadQueryForStruct(value, opt)
	if err != nil {
		return zero, err
	}
	if db == nil {
		return zero, ErrNilDB
	}
	agents := make([]*nullZeroAgent, 0)
	r, err := scan(db, queryStr, args, func() (T, []any) {
		dest, fields, ag := prepareScanDestinations(value, dests, opt)
		agents = append(agents, ag...)
		return dest, fields
	})
	if err != nil {
		return zero, err
	}
	if len(r) == 0 {
		return zero, sql.ErrNoRows
	}
	for _, agent := range agents {
		agent.Apply()
	}
	return value, nil
}

func buildLoadQueryForStruct[T any](value T, opt *Options) (query string, args []any, dests []fieldInfo, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	info, err := getStructInfo(value)
	if err != nil {
		return "", nil, nil, err
	}
	loadInfo, err := buildLoadInfo(opt.dialect, info, value)
	if err != nil {
		return "", nil, nil, err
	}

	conds := make([]sqlf.Builder, len(loadInfo.wheres))
	for i, col := range loadInfo.wheres {
		conds[i] = eqOrIsNull(col.Indent, col.Value)
	}

	b := sqlb.NewSelectBuilder().
		Select(util.Map(loadInfo.selects, func(c fieldData) sqlf.Builder {
			if c.Info.Load == "" {
				return sqlf.F(c.Indent)
			}
			return sqlf.F(c.Info.Load, sqlf.F(c.Indent))
		})...).
		From(sqlb.NewTable(loadInfo.table)).
		Where(sqlf.Join(" AND ", conds...))

	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, nil, err
	}
	if opt.debug {
		printDebugQuery("Load", value, query, args)
	}
	dests = util.Map(loadInfo.selects, func(i fieldData) fieldInfo {
		return i.Info
	})
	return query, args, dests, nil
}

type loadInfo struct {
	table   string
	selects []fieldData
	wheres  []fieldData
}

func buildLoadInfo[T any](dialect dialects.Dialect, f *structInfo, value T) (*loadInfo, error) {
	valueVal := reflect.ValueOf(value)
	if valueVal.Kind() == reflect.Ptr {
		valueVal = valueVal.Elem()
	}

	var (
		table         string
		selectColumns []fieldData
		whereColumns  []fieldData

		pk          fieldData
		unique      []fieldData
		constraints []fieldData
		match       []fieldData
	)
	for _, col := range f.columns {
		if col.Diving {
			continue
		}
		if table == "" && col.Table != "" {
			// respect first non-diving column with table specified
			table = col.Table
		}
		if col.Column == "" {
			continue
		}
		colIndent := dialect.QuoteIdentifier(col.Column)
		colValue, iszero, ok := getValueAtIndex(col.Index, valueVal)
		if !ok {
			return nil, fmt.Errorf("cannot get value for column %s", col.Column)
		}
		colData := fieldData{
			Info:   col,
			Indent: colIndent,
			Value:  colValue,
			IsZero: iszero,
		}

		switch {
		case col.PK:
			if pk.Indent != "" {
				return nil, errors.New("multiple primary key columns defined")
			}
			pk = colData
		case col.Unique && !iszero:
			unique = append(unique, colData)
		case col.ConflictOn:
			constraints = append(constraints, colData)
		case col.Match:
			// allow match columns to be zero-valued, like deleted_at = NULL
			match = append(match, colData)
		default:
			selectColumns = append(selectColumns, colData)
		}
	}
	if table == "" {
		return nil, errors.New("no table defined ")
	}
	if pk.Indent != "" && !pk.IsZero {
		whereColumns = append(whereColumns, pk)
		selectColumns = append(selectColumns, unique...)
		selectColumns = append(selectColumns, constraints...)
	} else if len(unique) > 0 {
		whereColumns = append(whereColumns, unique...)
		selectColumns = append(selectColumns, pk)
		selectColumns = append(selectColumns, constraints...)
	} else {
		allNonZero := len(constraints) > 0
		for _, v := range constraints {
			if v.Value == nil {
				allNonZero = false
				break
			}
		}
		if !allNonZero {
			return nil, errors.New("no primary field / unique field / conflict_on fields with non-zero values defined")
		}
		whereColumns = append(whereColumns, constraints...)
		selectColumns = append(selectColumns, pk)
		selectColumns = append(selectColumns, unique...)
	}
	whereColumns = append(whereColumns, match...)
	return &loadInfo{
		table:   table,
		selects: selectColumns,
		wheres:  whereColumns,
	}, nil
}
