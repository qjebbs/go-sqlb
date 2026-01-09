package mapper

import (
	"errors"
	"reflect"
)

// Blackhole is a scanner that drops the scanned value.
var Blackhole = &blackhole{}

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

type blackhole struct{}

func (b *blackhole) Scan(_ any) error { return nil }
