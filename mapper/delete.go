package mapper

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

// Delete Deletes a struct T from the database.
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"pk;col:id;table:users;"`
//
// The supported struct tags are:
//   - table<:name>: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields.
//   - col<:name>: The column in database associated with the field.
//   - pk: The column is primary key.
//   - unique: The column is unique.
//   - unique_group[:name[,name]...]: The column is one of the "Composite Unique" groups. If there is only one unique_group in the struct, the group name can be omitted.
//   - match: The column will be always included in WHERE clause even if it is zero value.
//   - soft_delete indicates this column is used for soft deletion. Supported types are *time.Time, sql.NullTime, and bool.
//
// It will return an error if it cannot locating a row to avoid accidental full-table delete.
// To locate the row, it will use non-zero `pk`, `unique`, or `unique_group` fields in priority order.
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

	var debugger *debugger
	if opt.debug {
		debugger = newDebugger("Delete", value, opt)
		defer debugger.print()
	}
	queryStr, args, err := buildDeleteQueryForStruct(value, opt)
	if err != nil {
		return err
	}
	if debugger != nil {
		debugger.onBuilt(queryStr, args)
	}
	if db == nil {
		return ErrNilDB
	}
	_, err = db.Exec(queryStr, args...)
	if debugger != nil {
		debugger.onExec(err)
	}
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
	deleteInfo, err := buildDeleteInfo(opt.dialect, info, value)
	if err != nil {
		return "", nil, err
	}

	if deleteInfo.softDelete.Indent != "" {
		switch v := deleteInfo.softDelete.Val.Raw.Interface().(type) {
		case *time.Time:
			// soft delete by setting current timestamp
			if deleteInfo.softDelete.Val.Value == nil {
				now := time.Now()
				deleteInfo.softDelete.Val.Value = &now
			}
		case sql.NullTime:
			if !v.Valid {
				// soft delete by setting current timestamp
				now := time.Now()
				deleteInfo.softDelete.Val.Value = sql.NullTime{Time: now, Valid: true}
			}
		case bool:
			// soft delete by setting true
			deleteInfo.softDelete.Val.Value = true
		default:
			return "", nil, fmt.Errorf("unsupported soft delete column %q with type %T", deleteInfo.softDelete.Info.Column, deleteInfo.softDelete.Val)
		}
		// build UPDATE ... SET <soft-delete col> = ? WHERE ...
		b := sqlb.NewUpdateBuilder().
			Update(deleteInfo.table).
			Set(deleteInfo.softDelete.Indent, deleteInfo.softDelete.Val.Value)
		deleteInfo.EachWhere(func(cond sqlf.Builder) {
			b.Where(cond)
		})
		query, args, err = b.BuildQuery(opt.style)
		if err != nil {
			return "", nil, err
		}
		return query, args, nil
	}

	b := sqlb.NewDeleteBuilder().
		DeleteFrom(deleteInfo.table)
	deleteInfo.EachWhere(func(cond sqlf.Builder) {
		b.Where(cond)
	})

	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, err
	}
	return query, args, nil
}

func buildDeleteInfo[T any](dialect sqlb.Dialect, f *structInfo, value T) (*locatingInfo, error) {
	valueVal := reflect.ValueOf(value)
	if valueVal.Kind() == reflect.Ptr {
		valueVal = valueVal.Elem()
	}
	return buildLocatingInfo(dialect, f, valueVal)
}
