package sqlb_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
)

func TestWithContextFunc(t *testing.T) {
	parent := sqlb.NewContext(context.Background(), dialect.PostgreSQL{})
	child := sqlb.ContextWithValue(parent, "k", "v")
	typeParent := reflect.TypeOf(parent)
	typeChild := reflect.TypeOf(child)

	if typeParent != typeChild {
		t.Fatalf("expected child context to have same type as parent: got %v, want %v", typeChild, typeParent)
	}
	if value := child.Value("k"); value != "v" {
		t.Fatalf("expected context value to be 'v': got %v", value)
	}
}
