package util

// Index returns the index of the first occurrence of v in s,
// or -1 if not present.
func Index[T comparable](s []T, v T) int {
	for i := range s {
		if v == s[i] {
			return i
		}
	}
	return -1
}

// IndexFunc returns the index of the first element in s
// that satisfies the predicate f, or -1 if none do.
func IndexFunc[T any](s []T, f func(T) bool) int {
	for i := range s {
		if f(s[i]) {
			return i
		}
	}
	return -1
}

// Concat concatenates multiple slices into one slice.
func Concat[T any](slices ...[]T) []T {
	totalLen := 0
	for _, s := range slices {
		totalLen += len(s)
	}
	result := make([]T, 0, totalLen)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// Map applies a function to each element of a slice and returns a new slice.
func Map[T1 any, T2 any](a []T1, f func(T1) T2) []T2 {
	if a == nil {
		return nil
	}
	b := make([]T2, len(a))
	for i, x := range a {
		b[i] = f(x)
	}
	return b
}

// MapKeys returns the keys of a map as a slice.
func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Filter filters a slice in-place based on a predicate function.
func Filter[T any](list []T, f func(T) bool) []T {
	pos := 0
	for _, x := range list {
		if f(x) {
			list[pos] = x
			pos++
		}
	}
	return list[:pos]
}
