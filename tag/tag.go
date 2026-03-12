package tag

// Info represents parsed tag information.
type Info struct {
	Model  bool   `json:",omitempty"` // Model indicates whether "model" key is present.
	Table  string `json:",omitempty"` // Table is parsed from "table" key.
	Column string `json:",omitempty"` // Column is parsed from "col" key.

	PK          bool    `json:",omitempty"` // PK indicates whether "pk" key is present.
	Required    bool    `json:",omitempty"` // Required indicates whether "required" key is present.
	ReadOnly    bool    `json:",omitempty"` // ReadOnly indicates whether "readonly" key is present.
	InsertZero  bool    `json:",omitempty"` // InsertZero indicates whether "insert_zero" key is present.
	Returning   bool    `json:",omitempty"` // Returning indicates whether "returning" key is present.
	ConflictOn  *string `json:",omitempty"` // ConflictOn indicates whether "conflict_on" key is present.
	ConflictSet *string `json:",omitempty"` // ConflictSet is parsed from "conflict_set" key.

	Unique       bool     `json:",omitempty"` // Unique indicates whether "unique" key is present.
	UniqueGroups []string `json:",omitempty"` // UniqueGroup indicates whether "unique_group" key is present.
	Match        bool     `json:",omitempty"` // Match indicates whether "match" key is present.

	SoftDelete bool `json:",omitempty"` // SoftDelete indicates whether "soft_delete" key is present.

	Select   string   `json:",omitempty"` // Select is parsed from "sel" key.
	From     []string `json:",omitempty"` // From is parsed from "from" key.
	SelectOn []string `json:",omitempty"` // On is parsed from "sel_on" key.
	Dive     bool     `json:",omitempty"` // Dive indicates whether "dive" key is present.
}
