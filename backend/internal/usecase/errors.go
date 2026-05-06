package usecase

import "errors"

var (
	ErrBadRequest = errors.New("bad request")
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrValidation = errors.New("validation error")
)

type AppError struct {
	Kind    error
	Message string
	Details map[string]any
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Kind.Error()
}

func (e *AppError) Unwrap() error {
	return e.Kind
}

func BadRequest(message string, details map[string]any) error {
	return &AppError{Kind: ErrBadRequest, Message: message, Details: details}
}

func NotFound(message string) error {
	return &AppError{Kind: ErrNotFound, Message: message}
}

func Conflict(message string, details map[string]any) error {
	return &AppError{Kind: ErrConflict, Message: message, Details: details}
}

func Validation(message string, details map[string]any) error {
	return &AppError{Kind: ErrValidation, Message: message, Details: details}
}
