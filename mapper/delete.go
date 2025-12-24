package mapper

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/qjebbs/go-sqlb"
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
//   - soft_delete indicates this column is used for soft deletion. Supported types are *time.Time, sql.NullTime, and bool.
//
// If a struct has all `pk`, `unique`, or `conflict_on` fields zero-valued, the `Delete()` operation will return an error.
// If all non-zero-valued, the priority for constructing the WHERE clause is `pk` > `unique` > `conflict_on`.
func Delete[T any](db QueryAble, value T, options ...Option) error {
	err := delete(db, value, options...)
	if err != nil {
		var zero T
		return wrapErrWithDebugName("Delete", zero, err)
	}
	return nil
}

func delete[T any](db QueryAble, value T, options ...Option) error {
	if err := checkPtrStruct(value); err != nil {
		return err
	}
	opt := mergeOptions(options...)
	queryStr, args, err := buildDeleteQueryForStruct(value, opt)
	if err != nil {
		return err
	}
	if db == nil {
		return ErrNilDB
	}
	_, err = db.Exec(queryStr, args...)
	return err
}

func buildDeleteQueryForStruct[T any](value T, opt *Options) (query string, args []any, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	info, err := getStructInfo(value)
	if err != nil {
		return "", nil, err
	}
	deleteInfo, err := buildLoadInfo(opt.dialect, info, value)
	if err != nil {
		return "", nil, err
	}

	if deleteInfo.softDel.Indent != "" {
		switch v := deleteInfo.softDel.Val.Raw.Interface().(type) {
		case *time.Time:
			// soft delete by setting current timestamp
			if deleteInfo.softDel.Val.Value == nil {
				now := time.Now()
				deleteInfo.softDel.Val.Value = &now
			}
		case sql.NullTime:
			if !v.Valid {
				// soft delete by setting current timestamp
				now := time.Now()
				deleteInfo.softDel.Val.Value = sql.NullTime{Time: now, Valid: true}
			}
		case bool:
			// soft delete by setting true
			deleteInfo.softDel.Val.Value = true
		default:
			return "", nil, fmt.Errorf("unsupported soft delete column %q with type %T", deleteInfo.softDel.Info.Column, deleteInfo.softDel.Val)
		}
		// build UPDATE ... SET <soft-delete col> = ? WHERE ...
		b := sqlb.NewUpdateBuilder().
			Update(deleteInfo.table).
			Set(deleteInfo.softDel.Indent, deleteInfo.softDel.Val.Value)
		for _, col := range deleteInfo.wheres {
			b.Where(eqOrIsNull(col.Indent, col.Val.Value))
		}
		query, args, err = b.BuildQuery(opt.style)
		if err != nil {
			return "", nil, err
		}
		if opt.debug {
			printDebugQuery("Delete", value, query, args)
		}
		return query, args, nil
	}

	b := sqlb.NewDeleteBuilder().
		DeleteFrom(deleteInfo.table)
	for _, col := range deleteInfo.wheres {
		b.Where(eqOrIsNull(col.Indent, col.Val.Value))
	}

	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, err
	}
	if opt.debug {
		printDebugQuery("Delete", value, query, args)
	}
	return query, args, nil
}
