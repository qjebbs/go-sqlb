package mapper

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/qjebbs/go-sqlf/v4/util"
)

func checkPtrStruct(value any) error {
	v := reflect.TypeOf(value)
	if v.Kind() != reflect.Ptr {
		return errors.New("value must be a pointer to struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errors.New("value must be a pointer to struct")
	}
	return nil
}

func checkStruct(value any) error {
	v := reflect.TypeOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	k := v.Kind()
	if k != reflect.Struct {
		return errors.New("value must be a struct or a pointer to struct")
	}
	return nil
}

func printDebugQuery(funcName string, value any, query string, args []any) {
	prefix := debugName(funcName, value)
	interpolated, err := util.Interpolate(query, args)
	if err != nil {
		fmt.Printf("[%s] interpolating: %s\n", prefix, err)
	}
	fmt.Printf("[%s] %s\n", prefix, interpolated)
}

func debugName(funcName string, value any) string {
	return fmt.Sprintf("%s(%T)", funcName, value)
}

func wrapErrWithDebugName(funcName string, value any, err error) error {
	if err == nil {
		return err
	}
	// not wrapping well known errors for easier checking
	if errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return fmt.Errorf("%s(%T): %w", funcName, value, err)
}

type valueInfo struct {
	// Raw reflect value
	Raw reflect.Value
	// Value of the field, no typed nil
	Value any
	// whether the value is zero value
	IsZero bool
}

func getValueAtIndex(dest []int, v reflect.Value) (*valueInfo, bool) {
	current, ok := getReflectValueAtIndex(dest, v)
	if !ok {
		return nil, false
	}
	r := &valueInfo{
		Raw:    current,
		Value:  current.Interface(),
		IsZero: current.IsZero(),
	}
	if current.Kind() == reflect.Ptr && current.IsNil() {
		// avoid typed nil
		r.Value = nil
	}
	return r, true
}

func getReflectValueAtIndex(dest []int, v reflect.Value) (reflect.Value, bool) {
	current := v
	for _, idx := range dest {
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return reflect.Value{}, false
			}
			current = current.Elem()
		}
		current = current.Field(idx)
	}
	return current, true
}
