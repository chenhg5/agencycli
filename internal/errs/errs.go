// Package errs defines sentinel error types for agencycli.
//
// These types are inspected by the CLI's main() to produce meaningful
// exit codes that agents can use for control-flow decisions:
//
//	0   success
//	1   general error
//	2   usage / bad arguments
//	3   resource not found
//	5   conflict / resource already exists
package errs

import "fmt"

// NotFoundError is returned when a requested resource does not exist.
// The CLI exits with code 3 when this error reaches main().
type NotFoundError struct {
	Resource string
	Name     string
}

func (e *NotFoundError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("%s %q not found", e.Resource, e.Name)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// NotFound returns a *NotFoundError for the given resource and name.
func NotFound(resource, name string) error {
	return &NotFoundError{Resource: resource, Name: name}
}

// ConflictError is returned when a resource already exists and the
// caller did not request an upsert / --if-not-exists behaviour.
// The CLI exits with code 5 when this error reaches main().
type ConflictError struct {
	Resource string
	Name     string
}

func (e *ConflictError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("%s %q already exists", e.Resource, e.Name)
	}
	return fmt.Sprintf("%s already exists", e.Resource)
}

// Conflict returns a *ConflictError for the given resource and name.
func Conflict(resource, name string) error {
	return &ConflictError{Resource: resource, Name: name}
}

// UsageError wraps a message that represents a caller mistake (wrong flags,
// missing arguments, etc.). The CLI exits with code 2 for these.
type UsageError struct {
	Msg string
}

func (e *UsageError) Error() string { return e.Msg }

// Usage returns a *UsageError with the given message.
func Usage(msg string) error { return &UsageError{Msg: msg} }
