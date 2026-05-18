package github

import (
	"errors"
	"fmt"
)

type ErrorKind string

const (
	ErrorKindNetwork    ErrorKind = "network"
	ErrorKindPermission ErrorKind = "permission"
	ErrorKindUnknown    ErrorKind = "unknown"
)

type NotFoundError struct {
	Repo   Repository
	Number int
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("github issue %s#%d not found", e.Repo.String(), e.Number)
}

type RequestError struct {
	Kind      ErrorKind
	Operation string
	Err       error
}

func (e RequestError) Error() string {
	if e.Operation == "" {
		return fmt.Sprintf("github %s error: %v", e.Kind, e.Err)
	}
	return fmt.Sprintf("github %s error during %s: %v", e.Kind, e.Operation, e.Err)
}

func (e RequestError) Unwrap() error {
	return e.Err
}

func IsNotFound(err error) bool {
	var target NotFoundError
	return errors.As(err, &target)
}

func IsRequestKind(err error, kind ErrorKind) bool {
	var target RequestError
	return errors.As(err, &target) && target.Kind == kind
}
