package generate

import (
	"go/types"
)

// findUnderlyingStruct recursively resolves types to find the base struct.
// It operates purely on go/types information.
func findUnderlyingStruct(t types.Type) *types.Struct {
	if t == nil {
		return nil
	}

	// If it's already a struct, we're done.
	if s, ok := t.Underlying().(*types.Struct); ok {
		return s
	}

	// Follow named types and aliases.
	for {
		switch next := t.(type) {
		case *types.Named:
			t = next.Underlying()
		case *types.Alias:
			t = next.Rhs()
		default:
			// Not a type we can resolve further.
			return nil
		}

		// Check if the new type is a struct.
		if s, ok := t.Underlying().(*types.Struct); ok {
			return s
		}

		// If the underlying type is the same as the current type, we've hit a loop or a base type.
		if t == t.Underlying() {
			return nil
		}
	}
}
