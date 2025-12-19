# Tag syntax for sqlb mapper

The `sqlb` tag syntax is:

```
key[:value][;key[:value]]...
```

For example, `sqlb:"pk;col:id;from:u;"`

## Terminology

- Inheritance: Some keys can be inherited by the subsequent sibling fields and by the sub-fields of the structure.
- Applied Table Name: The name of the table that is effective in the current query. For example, `f` in `sqlb.NewTable("foo", "f")`, and `foo` in `sqlb.NewTable("foo")`.

## Supported keys

For SELECT operations, the supported keys are:

key|Inheritance|description
---|---|---
table|Yes|Declare base table for the field or its sub-fields / subsequent sibling fields. It usually works with `WithNullZeroTables()` Option.
sel|No|Specify expression to select for this field. It's used together with `from` key to declare tables used in the expression, e.g. `sel:COALESCE(?.name,'');from:u;`, which is required by dependency analysis.
col|No|If `sel` key is not specified, specify the column to select for this field. It's recommended to use `col` key for simple column selection, which can be shared usage in INSERT/UPDATE operations. e.g. `col:name;from:u;`
from|Yes|Declare from tables for this field or its sub-fields / subsequent sibling fields. It accepts multiple **Applied Table Name**, comma-separated, e.g. `from:f,b`.
on|No|Scan the field only on any one of tags specified, comma-separated. e.g. `on:full;`
dive|No|For struct fields, dive into scan its field. e.g. `dive;`


For INSERT operations, the supported keys are:

key|Inheritance|description
---|---|---
table|Yes|Declare base table for the field or its sub-fields / subsequent sibling fields.
col|No|Specify the column associated with the field.
returning|No|Mark the field to be included in RETURNING clause.
conflict_on|No|Declare current as one of conflict detection column.
conflict_set|No|Update the field on conflict. It's equivalent to `SET <column>=EXCLUDED.<column>` in ON CONFLICT clause if not specified with value, and can be specified with expression, e.g. `conflict_set:NULL`, which is equivalent to `SET <column>=NULL`.

For UPDATE operations, the supported keys are:

key|Inheritance|description
---|---|---
table|Yes|Declare base table for the field or its sub-fields / subsequent sibling fields.
col|No|Specify the column associated with the field.
noupdate|No|Mark the field to be excluded from UPDATE statement.
pk|No|Mark the field as primary key, which will be used in WHERE clause.
match|No|Mark the field as match column, which will be used in WHERE clause.
