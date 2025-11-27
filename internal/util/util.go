package util

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
