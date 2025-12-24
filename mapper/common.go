package mapper

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
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

func getValueAtIndex(dest []int, v reflect.Value) (value any, iszero, ok bool) {
	current, ok := getReflectValueAtIndex(dest, v)
	if !ok {
		return nil, false, false
	}
	if current.Kind() == reflect.Ptr && current.IsNil() {
		// avoid typed nil
		return nil, true, true
	}
	return current.Interface(), current.IsZero(), true
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
