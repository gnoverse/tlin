package fixerv2

import "regexp"

// Option represents a container type for handling
// values with potential errors
type Option[T any] struct {
	value T
	err   error
}

// Result holds the compile regex and its captured group mappings
type Result struct {
	regex    *regexp.Regexp
	captures map[string]int
}

// createOption creates a new Option
func createOption[T any](value T, err error) Option[T] {
	return Option[T]{value: value, err: err}
}

// Map applies a function to the Option value
func (o Option[T]) Map(f func(T) T) Option[T] {
	if o.err != nil {
		return o
	}
	return createOption(f(o.value), nil)
}

// Bind chains Option operations while handling potential errors
func (o Option[T]) Bind(f func(T) Option[T]) Option[T] {
	if o.err != nil {
		return o
	}
	return f(o.value)
}
