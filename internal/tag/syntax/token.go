package syntax

type token struct {
	typ   TokenType
	lit   string
	bad   bool
	start int
	end   int
	pos   Pos
}

// TokenType is the type of token.
type TokenType string

const (
	_Error TokenType = "Error"

	_Key       = "Key"
	_Colon     = ":"
	_Value     = "Value"
	_Semicolon = ";"

	_EOF = "EOF"
)
