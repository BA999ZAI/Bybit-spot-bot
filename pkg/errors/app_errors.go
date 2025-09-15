package errors

import (
	"fmt"
	"net/http"
)

type ErrorType string

const (
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeDomain     ErrorType = "domain"
	ErrorTypeExternal   ErrorType = "external"
	ErrorTypeInternal   ErrorType = "internal"
	ErrorTypeNotFound   ErrorType = "not_found"
)

type AppError struct {
	Type    ErrorType              `json:"type"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func ValidationError(field, message string) *AppError {
	return &AppError{
		Type:    ErrorTypeValidation,
		Code:    "VALIDATION_ERROR",
		Message: fmt.Sprintf("validation failed for field '%s': %s", field, message),
		Details: map[string]interface{}{
			"field": field,
		},
	}
}

func DomainError(message, code string) *AppError {
	return &AppError{
		Type:    ErrorTypeDomain,
		Code:    code,
		Message: message,
	}
}

func ExternalError(service, message string) *AppError {
	return &AppError{
		Type:    ErrorTypeExternal,
		Code:    "EXTERNAL_SERVICE_ERROR",
		Message: fmt.Sprintf("external service '%s' error: %s", service, message),
		Details: map[string]interface{}{
			"service": service,
		},
	}
}

func InternalError(message string) *AppError {
	return &AppError{
		Type:    ErrorTypeInternal,
		Code:    "INTERNAL_ERROR",
		Message: message,
	}
}

func NotFoundError(resource, identifier string) *AppError {
	return &AppError{
		Type:    ErrorTypeNotFound,
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s with identifier '%s' not found", resource, identifier),
		Details: map[string]interface{}{
			"resource":   resource,
			"identifier": identifier,
		},
	}
}

func (e *AppError) GetHTTPStatus() int {
	switch e.Type {
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeDomain:
		return http.StatusUnprocessableEntity
	case ErrorTypeExternal:
		return http.StatusBadGateway
	case ErrorTypeInternal:
		return http.StatusInternalServerError
	case ErrorTypeNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
