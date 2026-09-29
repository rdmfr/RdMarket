package errors

import "fmt"

type ErrorCode string

const (
	ErrInternalError       ErrorCode = "INTERNAL_ERROR"
	ErrBadRequest          ErrorCode = "BAD_REQUEST"
	ErrUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrForbidden           ErrorCode = "FORBIDDEN"
	ErrNotFound            ErrorCode = "NOT_FOUND"
	ErrUnprocessableEntity ErrorCode = "UNPROCESSABLE_ENTITY"
	ErrRateLimited         ErrorCode = "RATE_LIMITED"
	ErrServiceUnavailable  ErrorCode = "SERVICE_UNAVAILABLE"
	ErrInvalidCredentials  ErrorCode = "INVALID_CREDENTIALS"
	ErrCSRFInvalid         ErrorCode = "CSRF_INVALID"
	ErrInsufficientData    ErrorCode = "INSUFFICIENT_DATA"
	ErrSimulationForbidden ErrorCode = "SIMULATION_FORBIDDEN"
)

type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func (c ErrorCode) String() string {
	return string(c)
}
