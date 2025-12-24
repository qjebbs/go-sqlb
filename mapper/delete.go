package mapper

import (
	"database/sql"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// Delete Deletes a struct T from the database.
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"pk;col:id;table:users;"`
//
// The supported struct tags are:
//   - table: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields.
//   - col: The column associated with the field.
//   - pk: The column is primary key, which could be used in WHERE clause to locate the row.
//   - unique: The column could be used in WHERE clause to locate the row. There's no tag for "Composite Unique" fields, since any one of them is not unique alone.
//   - conflict_on: Multiple of them form a composite unique constraint, which could be used in WHERE clause to locate the row.
//   - match: The column will be always included in WHERE clause even if it is zero value.
//
// If a struct has all `pk`, `unique`, or `conflict_on` fields zero-valued, the `Delete()` operation will return an error.
// If all non-zero-valued, the priority for constructing the WHERE clause is `pk` > `unique` > `conflict_on`.
func Delete[T any](db QueryAble, value T, options ...Option) (T, error) {
	r, err := delete(db, value, options...)
	if err != nil {
		var zero T
		return zero, wrapErrWithDebugName("Delete", zero, err)
	}
	return r, nil
}

func delete[T any](db QueryAble, value T, options ...Option) (T, error) {
	var zero T
	if err := checkPtrStruct(value); err != nil {
		return zero, err
	}
	opt := mergeOptions(options...)
	queryStr, args, dests, err := buildDeleteQueryForStruct(value, opt)
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

func buildDeleteQueryForStruct[T any](value T, opt *Options) (query string, args []any, dests []fieldInfo, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	info, err := getStructInfo(value)
	if err != nil {
		return "", nil, nil, err
	}
	deleteInfo, err := buildLoadInfo(opt.dialect, info, value)
	if err != nil {
		return "", nil, nil, err
	}

	conds := make([]sqlf.Builder, len(deleteInfo.wheres))
	for i, col := range deleteInfo.wheres {
		conds[i] = eqOrIsNull(col.Indent, col.Value)
	}

	b := sqlb.NewDeleteBuilder().
		DeleteFrom(deleteInfo.table).
		Where(sqlf.Join(" AND ", conds...))

	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, nil, err
	}
	if opt.debug {
		printDebugQuery("Delete", value, query, args)
	}
	dests = util.Map(deleteInfo.selects, func(i fieldData) fieldInfo {
		return i.Info
	})
	return query, args, dests, nil
}
