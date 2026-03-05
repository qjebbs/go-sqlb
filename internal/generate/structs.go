package generate

// Info holds information about the package and structs to be used in the code generation template.
type Info struct {
	Name    string
	Structs []StructInfo
}

// StructInfo holds information about a struct to be processed by the template.
type StructInfo struct {
	Name        string
	Columns     []ColumnInfo
	UniqueTable string
}

// ColumnInfo holds information about a struct field that maps to a database column.
type ColumnInfo struct {
	FieldName  string
	TableName  string
	ColumnName string
}

// FieldInfo is a helper struct to pass field information around.
// It can represent a field from either AST or go/types.
type FieldInfo struct {
	Name        string
	Tag         string
	IsAnonymous bool
	IsExported  bool
	Type        interface{} // Can be ast.Expr or types.Type
}
