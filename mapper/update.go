package mapper

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

// Update updates a single struct into the database.
//
// Update omits zero-value fields by default. To force do a full update including zero-value fields,
// use WithUpdateAll() option.
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"pk;col:id;table:users;"`
//
// The supported struct tags are:
//   - table<:name>: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields.
//   - col<:name>: The column in database associated with the field.
//   - pk: The column is primary key.
//   - required: The field is required to have non-zero value, otherwise Update (not Patch) will return an error.
//   - unique: The column is unique.
//   - unique_group[:name[,name]...]: The column is one of the "Composite Unique" groups. If there is only one unique_group in the struct, the group name can be omitted.
//   - match: The column will be always included in WHERE clause even if it is zero value.
//   - readonly: The field is excluded from UPDATE statement.
//
// It will return an error if it cannot locating a row to avoid accidental full-table update.
// To locate the row, it will use non-zero `pk`, `unique`, or `unique_group` fields in priority order.
func Update[T any](ctx *sqlf.Context, db QueryAble, value T, options ...Option) error {
	return wrapErrWithDebugName("Update", value, update(ctx, db, value, true, options...))
}

// Patch is similar to Update(), but it only updates non-zero fields of the struct.
//
// See Update() for more details.
func Patch[T any](ctx *sqlf.Context, db QueryAble, value T, options ...Option) error {
	return wrapErrWithDebugName("Patch", value, update(ctx, db, value, false, options...))
}

func update[T any](ctx *sqlf.Context, db QueryAble, value T, updateAll bool, options ...Option) error {
	if err := checkStruct(value); err != nil {
		return err
	}
	opt := mergeOptions(options...)
	var debugger *debugger
	if opt.debug {
		if updateAll {
			debugger = newDebugger("Update", value, opt)
		} else {
			debugger = newDebugger("Patch", value, opt)
		}
		defer debugger.print(ctx.Dialect())
	}
	queryStr, args, err := buildUpdateQueryForStruct(ctx, value, updateAll, opt)
	if err != nil {
		return err
	}
	if debugger != nil {
		debugger.onBuilt(queryStr, args)
	}
	if db == nil {
		return ErrNilDB
	}
	r, err := db.Exec(queryStr, args...)
	if err != nil {
		return err
	}
	if debugger != nil {
		debugger.onExec(err)
	}
	rowsAffected, err := r.RowsAffected()
	if err != nil {
		return nil
	}
	if rowsAffected == 0 {
		return errors.New("no rows updated")
	}
	if rowsAffected > 1 {
		return fmt.Errorf("unexpectedly updated %d rows, wrong 'pk'?", rowsAffected)
	}
	return err
}

func buildUpdateQueryForStruct[T any](ctx *sqlf.Context, value T, updateAll bool, opt *Options) (query string, args []any, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}

	b := sqlb.NewUpdateBuilder()

	var zero T
	info, err := getStructInfo(zero)
	if err != nil {
		return "", nil, err
	}
	updateInfo, err := buildUpdateInfo(info, updateAll, value)
	if err != nil {
		return "", nil, err
	}

	if updateInfo.table != "" {
		// don't override with empty table in case the table is set manually
		b.Update(updateInfo.table)
	}
	for _, coldata := range updateInfo.updateColumns {
		b.Set(coldata.Info.Column, coldata.Val.Value)
	}

	updateInfo.EachWhere(func(cond sqlf.Builder) {
		b.Where(cond)
	})

	query, args, err = b.Build(ctx)
	if err != nil {
		return "", nil, err
	}
	return query, args, nil
}

type updateInfo struct {
	locatingInfo
	updateColumns []fieldData
}

func buildUpdateInfo[T any](f *structInfo, updateAll bool, value T) (*updateInfo, error) {
	valueVal := reflect.ValueOf(value)
	if valueVal.Kind() == reflect.Ptr {
		valueVal = valueVal.Elem()
	}
	locatingInfo, err := buildLocatingInfo(f, valueVal)
	if err != nil {
		return nil, err
	}
	var updateColumns []fieldData

	for _, col := range locatingInfo.others {
		if col.Info.PK || col.Info.ReadOnly || col.Info.SoftDelete || (col.Val.IsZero && !updateAll) {
			continue
		}
		if col.Info.Required && col.Val.IsZero {
			return nil, fmt.Errorf("%s is required", col.Info.Name)
		}
		updateColumns = append(updateColumns, col)
	}
	if len(updateColumns) == 0 {
		return nil, errors.New("no updatable columns found for update")
	}
	return &updateInfo{
		locatingInfo:  *locatingInfo,
		updateColumns: updateColumns,
	}, nil
}
