# Tag syntax for struct scanning

The `sqlb` tag syntax is:

```
key:value[;key:value]...
```

Supported keys:
  - `col`: specifies the column to select for this field. (required)
  - `on`: scan the field only on any one of tags specified, comma-separated. (optional)
  - `tables`: available only for anonymous fields, declare tables for fields of the anonymous struct, comma-separated. (optional)

The column definition supports two formats:
 1. `<table>.<column>`
 2. `<expression>|[table1, table2...]`

The tables declare in format 2 is optional, since,
 1. The expression could use no table.
 2. The expression could use tables from those declared in parent anonymous struct. (As `Model` in the example below)

Example:

```go
type Model struct {
    ID   int    `sqlb:"sel:?.id"`
}

type User struct {
    Model    `sqlb:"tables:u"`  Declare tables for its fields
    Name     string `sqlb:"u.name"` // Simple syntax
    Age      int    `sqlb:"sel:COALESCE(?.age,0);tables:u"` // Equals to sqlf.F("COALESCE(?.age,0)", u)
    Settings string `sqlb:"sel:u.name;on:full"` // Scanned only when "full" tag specified
}

var Users = sqlb.NewTable("users", "u")
b := sqlb.NewSelectBuilder().
    From(Users).
    Where(Users.Column("active")))

users, err := sqlb.Select[*User](db, b, scanner.WithBindStyle(sqlf.Question), scanner.WithTag("full"))
if err != nil {
    handle error
}
```