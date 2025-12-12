package syntax

import "fmt"

// A Pos represents an absolute (line, col) source position
type Pos struct {
	line, col uint
}

// NewPos returns a new Pos for the given line and column.
func NewPos(line, col uint) Pos { return Pos{line, col} }

// Line returns the line number of the position.
func (p Pos) Line() uint { return p.line }

// Col returns the column number of the position.
func (p Pos) Col() uint { return p.col }

func (p Pos) String() string {
	if p.line == 0 {
		return "<unknown position>"
	}
	if p.col == 0 {
		return fmt.Sprintf("%d", p.line)
	}
	return fmt.Sprintf("%d:%d", p.line, p.col)
}
