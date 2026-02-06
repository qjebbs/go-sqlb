package mapper

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

type locatingInfo struct {
	table string

	locatingColumn []fieldData // from pk or unique (single) / unique_group (composite)
	matchColumns   []fieldData // from match
	softDelete     *fieldData  // from soft_delete
	others         []fieldData // other columns
}

type fieldData struct {
	Info fieldInfo
	Val  valueInfo
}

func (i locatingInfo) EachWhere(fn func(cond sqlf.Builder)) {
	for _, coldata := range i.locatingColumn {
		fn(eqOrIsNull(coldata.Info.NewColumnBuilder(), coldata.Val.Value))
	}
	for _, coldata := range i.matchColumns {
		fn(eqOrIsNull(coldata.Info.NewColumnBuilder(), coldata.Val.Value))
	}
}

func eqOrIsNull(column sqlf.Builder, value any) sqlf.Builder {
	if value == nil {
		return sqlf.F("? IS NULL", column)
	}
	return sqlf.F("? = ?", column, value)
}

func buildLocatingInfo(f *structInfo, value reflect.Value) (*locatingInfo, error) {
	var (
		r                  locatingInfo
		seenPK             bool
		pk                 fieldData
		seenSoftDelete     bool
		uniqueColumns      []fieldData
		uniqueGroupColumns []fieldData
		uniqueGroups       = make(map[string][]fieldData)
		invalidGroups      = make(map[string]bool)

		uniqueNames []string
	)
	for _, col := range f.columns {
		if col.Diving {
			continue
		}
		if r.table == "" && col.Table != "" {
			// respect first column with table specified
			r.table = col.Table
		}
		if col.Column == "" {
			continue
		}
		colValue, ok := getValueAtIndex(col.Index, value)
		if !ok {
			return nil, fmt.Errorf("cannot get value for column %s", col.Column)
		}
		data := fieldData{
			Info: col,
			Val:  *colValue,
		}
		if col.PK {
			if seenPK {
				return nil, errors.New("multiple primary key columns defined")
			}
			seenPK = true
			uniqueNames = append(uniqueNames, col.Name)
			if !data.Val.IsZero {
				pk = data
			}
		}
		if col.SoftDelete {
			// soft delete column will be used in WHERE clause,
			// since we should report error when updating a soft-deleted row.
			if seenSoftDelete {
				return nil, errors.New("multiple soft delete columns defined")
			}
			seenSoftDelete = true
			r.softDelete = &data
			// copy and set zero value to match condition:
			// WHERE <soft-delete> = <zero>
			cp := data
			if !cp.Val.IsZero {
				rv := reflect.Zero(cp.Val.Raw.Type())
				if rv.Kind() == reflect.Ptr && rv.IsNil() {
					// avoid typed nil
					cp.Val.Value = nil
				} else {
					cp.Val.Value = rv.Interface()
				}
			}
			r.matchColumns = append(r.matchColumns, cp)
		}
		if col.Unique {
			if !colValue.IsZero {
				uniqueColumns = append(uniqueColumns, data)
			}
			uniqueNames = append(uniqueNames, col.Name)
		}
		if len(col.UniqueGroups) > 0 {
			uniqueGroupColumns = append(uniqueGroupColumns, data)
			for _, d := range col.UniqueGroups {
				uniqueGroups[d] = append(uniqueGroups[d], data)
			}
			if colValue.IsZero {
				// if any column in the unique group is zero, the whole group is invalid
				for _, d := range col.UniqueGroups {
					invalidGroups[d] = true
				}
			}
		}
		if col.Match {
			r.matchColumns = append(r.matchColumns, data)
		}
		// add all, remove locating columns later
		r.others = append(r.others, data)
	}
	switch {
	case pk.Info.Column != "":
		r.locatingColumn = []fieldData{pk}
	case len(uniqueColumns) > 0:
		if len(uniqueColumns) > 1 {
			return nil, errors.New("multiple unique columns with values defined, cannot locate the row")
		}
		// use the only unique column as uniqueColumn
		r.locatingColumn = []fieldData{uniqueColumns[0]}
	default:
		// try unique groups
		found := false
		for groupName, cols := range uniqueGroups {
			if invalidGroups[groupName] {
				continue
			}
			if found {
				return nil, errors.New("multiple unique groups with values defined, cannot locate the row")
			}
			found = true
			r.locatingColumn = append(r.locatingColumn, cols...)
		}
		if !found {
			sb := new(strings.Builder)
			for i, name := range uniqueNames {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(name)
			}
			for _, g := range uniqueGroups {
				if sb.Len() > 0 {
					sb.WriteString(", ")
				}
				names := util.Map(g, func(f fieldData) string { return f.Info.Name })
				sb.WriteString(strings.Join(names, "+"))
			}
			return nil, errors.New("require either of to locate a row: " + sb.String())
		}
	}
	r.others = util.Filter(r.others, func(col fieldData) bool {
		for _, locatingCol := range r.locatingColumn {
			if col.Info.Column == locatingCol.Info.Column {
				return false
			}
		}
		for _, locatingCol := range r.matchColumns {
			if col.Info.Column == locatingCol.Info.Column {
				return false
			}
		}
		return true
	})
	return &r, nil
}
