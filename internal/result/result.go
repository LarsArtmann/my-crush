package result

// Result represents a type-safe operation result that can either succeed or fail
// This eliminates the need for returning (value, error) tuples everywhere
type Result[T, E any] struct {
	value T
	err   E
	isOk bool
}

// Ok creates a successful result
func Ok[T, E any](value T) Result[T, E] {
	return Result[T, E]{
		value: value,
		err:   *new(E), // Zero value for error type
		isOk: true,
	}
}

// Err creates an error result
func Err[T, E any](err E) Result[T, E] {
	return Result[T, E]{
		value: *new(T), // Zero value for success type
		err:   err,
		isOk: false,
	}
}

// IsSuccess returns true if the result is successful
func (r Result[T, E]) IsSuccess() bool {
	return r.isOk
}

// IsError returns true if the result is an error
func (r Result[T, E]) IsError() bool {
	return !r.isOk
}

// Value returns the success value (panics if called on error result)
func (r Result[T, E]) Value() T {
	if !r.isOk {
		panic("Cannot get value from error result - call IsError() first")
	}
	return r.value
}

// Error returns the error (panics if called on success result)
func (r Result[T, E]) Error() E {
	if r.isOk {
		panic("Cannot get error from success result - call IsSuccess() first")
	}
	return r.err
}

// Unwrap returns the traditional (value, error) tuple for compatibility
func (r Result[T, E]) Unwrap() (T, E) {
	return r.value, r.err
}

// Map applies a function to the success value, preserving error state
func Map[T, E, U any](r Result[T, E], fn func(T) U) Result[U, E] {
	if r.isOk {
		return Ok[U, E](fn(r.value))
	}
	return Err[U, E](r.err)
}

// MapErr applies a function to the error, preserving success state
func MapErr[T, E, F any](r Result[T, E], fn func(E) F) Result[T, F] {
	if r.isOk {
		return Ok[T, F](r.value)
	}
	return Err[T, F](fn(r.err))
}

// Match executes different functions based on success or error state
func Match[T, E, U any](r Result[T, E], onSuccess func(T) U, onError func(E) U) U {
	if r.isOk {
		return onSuccess(r.value)
	}
	return onError(r.err)
}