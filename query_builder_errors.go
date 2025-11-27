package sqlb

import (
	"errors"
	"strings"
)

func (b *QueryBuilder) pushError(err error) {
	b.errors = append(b.errors, err)
}

func (b *QueryBuilder) anyError() error {
	if len(b.errors) == 0 {
		return nil
	}
	sb := new(strings.Builder)
	sb.WriteString("collected errors: \n")
	for _, err := range b.errors {
		sb.WriteString(" - ")
		sb.WriteString(err.Error())
		sb.WriteRune('\n')
	}
	return errors.New(sb.String())
}
