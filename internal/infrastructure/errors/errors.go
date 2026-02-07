package errors

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error types for different scenarios
type ErrorType string

const (
	
	ErrorTypeValidation   ErrorType = "VALIDATION_ERROR"
	ErrorTypeNotFound     ErrorType = "NOT_FOUND"
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden    ErrorType = "FORBIDDEN"
	ErrorTypeInternal     ErrorType = "INTERNAL_ERROR"
	ErrorTypeBadRequest   ErrorType = "BAD_REQUEST"
	ErrorTypeConflict     ErrorType = "CONFLICT"
	ErrorTypeTimeout      ErrorType = "TIMEOUT"
	ErrorTypeUnavailable  ErrorType = "UNAVAILABLE"
	ErrorTypeRateLimited  ErrorType = "RATE_LIMITED"
)

// ServiceError represents a structured service error
type ServiceError struct {
	Type        ErrorType `json:"type"`
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Details     string    `json:"details,omitempty"`
	RequestID   string    `json:"request_id,omitempty"`
	ServiceName string    `json:"service_name,omitempty"`
	Timestamp   string    `json:"timestamp,omitempty"`
	Err         error     `json:"-"`
}

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Err.Error())
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

// Error constructors
func NewValidationError(message, details string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeValidation,
		Code:    "VALIDATION_FAILED",
		Message: message,
		Details: details,
	}
}

func NewNotFoundError(resource, id string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeNotFound,
		Code:    "RESOURCE_NOT_FOUND",
		Message: fmt.Sprintf("%s with id %s not found", resource, id),
		Details: fmt.Sprintf("Resource: %s, ID: %s", resource, id),
	}
}

func NewUnauthorizedError(message string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeUnauthorized,
		Code:    "UNAUTHORIZED",
		Message: message,
	}
}

func NewForbiddenError(message string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeForbidden,
		Code:    "FORBIDDEN",
		Message: message,
	}
}

func NewInternalError(message string, err error) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeInternal,
		Code:    "INTERNAL_ERROR",
		Message: message,
		Err:     err,
	}
}

func NewBadRequestError(message, details string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeBadRequest,
		Code:    "BAD_REQUEST",
		Message: message,
		Details: details,
	}
}

func NewConflictError(message, details string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeConflict,
		Code:    "CONFLICT",
		Message: message,
		Details: details,
	}
}

func NewTimeoutError(message string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeTimeout,
		Code:    "TIMEOUT",
		Message: message,
	}
}

func NewUnavailableError(message string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeUnavailable,
		Code:    "SERVICE_UNAVAILABLE",
		Message: message,
	}
}

func NewRateLimitedError(message string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeRateLimited,
		Code:    "RATE_LIMITED",
		Message: message,
	}
}

// Convert ServiceError to gRPC status
func ToGRPCStatus(err *ServiceError) *status.Status {
	var grpcCode codes.Code
	var grpcMessage string

	switch err.Type {
	case ErrorTypeValidation:
		grpcCode = codes.InvalidArgument
		grpcMessage = err.Message
	case ErrorTypeNotFound:
		grpcCode = codes.NotFound
		grpcMessage = err.Message
	case ErrorTypeUnauthorized:
		grpcCode = codes.Unauthenticated
		grpcMessage = err.Message
	case ErrorTypeForbidden:
		grpcCode = codes.PermissionDenied
		grpcMessage = err.Message
	case ErrorTypeBadRequest:
		grpcCode = codes.InvalidArgument
		grpcMessage = err.Message
	case ErrorTypeConflict:
		grpcCode = codes.AlreadyExists
		grpcMessage = err.Message
	case ErrorTypeTimeout:
		grpcCode = codes.DeadlineExceeded
		grpcMessage = err.Message
	case ErrorTypeUnavailable:
		grpcCode = codes.Unavailable
		grpcMessage = err.Message
	case ErrorTypeRateLimited:
		grpcCode = codes.ResourceExhausted
		grpcMessage = err.Message
	default:
		grpcCode = codes.Internal
		grpcMessage = "Internal server error"
	}

	st := status.New(grpcCode, grpcMessage)

	// Add error details as metadata
	if err.Details != "" {
		details := &ErrorDetails{
			Type:      string(err.Type),
			Code:      err.Code,
			Details:   err.Details,
			RequestID: err.RequestID,
		}
		st, _ = st.WithDetails(details)
	}

	return st
}

// Convert any error to ServiceError
func FromError(err error) *ServiceError {
	if err == nil {
		return nil
	}

	// Check if it's already a ServiceError
	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr
	}

	// Check if it's a gRPC status error
	if st, ok := status.FromError(err); ok {
		return fromGRPCStatus(st)
	}

	// Default to internal error
	return NewInternalError("An unexpected error occurred", err)
}

// Convert gRPC status to ServiceError
func fromGRPCStatus(st *status.Status) *ServiceError {
	var errorType ErrorType
	var code string

	switch st.Code() {
	case codes.InvalidArgument:
		errorType = ErrorTypeValidation
		code = "VALIDATION_FAILED"
	case codes.NotFound:
		errorType = ErrorTypeNotFound
		code = "RESOURCE_NOT_FOUND"
	case codes.Unauthenticated:
		errorType = ErrorTypeUnauthorized
		code = "UNAUTHORIZED"
	case codes.PermissionDenied:
		errorType = ErrorTypeForbidden
		code = "FORBIDDEN"
	case codes.AlreadyExists:
		errorType = ErrorTypeConflict
		code = "CONFLICT"
	case codes.DeadlineExceeded:
		errorType = ErrorTypeTimeout
		code = "TIMEOUT"
	case codes.Unavailable:
		errorType = ErrorTypeUnavailable
		code = "SERVICE_UNAVAILABLE"
	case codes.ResourceExhausted:
		errorType = ErrorTypeRateLimited
		code = "RATE_LIMITED"
	default:
		errorType = ErrorTypeInternal
		code = "INTERNAL_ERROR"
	}

	return &ServiceError{
		Type:    errorType,
		Code:    code,
		Message: st.Message(),
	}
}

// Error logging helper
func LogError(ctx context.Context, logger *zap.Logger, err error, fields ...zap.Field) {
	serviceErr := FromError(err)

	// Add error-specific fields
	errorFields := []zap.Field{
		zap.String("error_type", string(serviceErr.Type)),
		zap.String("error_code", serviceErr.Code),
		zap.String("error_message", serviceErr.Message),
	}

	if serviceErr.Details != "" {
		errorFields = append(errorFields, zap.String("error_details", serviceErr.Details))
	}

	if serviceErr.RequestID != "" {
		errorFields = append(errorFields, zap.String("request_id", serviceErr.RequestID))
	}

	// Add original error if different from service error
	if serviceErr.Err != nil && serviceErr.Err != err {
		errorFields = append(errorFields, zap.Error(serviceErr.Err))
	}

	// Add additional fields
	errorFields = append(errorFields, fields...)

	logger.Error("Service error occurred", errorFields...)
}

// Validation error helper
func NewValidationErrorFromValidator(err error) *ServiceError {
	if err == nil {
		return nil
	}

	// Extract validation error details
	details := err.Error()
	message := "Validation failed"

	// Try to extract field-specific errors
	if strings.Contains(details, "validation failed") {
		message = "One or more fields failed validation"
	}

	return NewValidationError(message, details)
}

// Error details for gRPC metadata
type ErrorDetails struct {
	Type      string `protobuf:"bytes,1,opt,name=type,proto3" json:"type"`
	Code      string `protobuf:"bytes,2,opt,name=code,proto3" json:"code"`
	Details   string `protobuf:"bytes,3,opt,name=details,proto3" json:"details,omitempty"`
	RequestID string `protobuf:"bytes,4,opt,name=request_id,json=requestId,proto3" json:"request_id,omitempty"`
}

func (m *ErrorDetails) Reset()         { *m = ErrorDetails{} }
func (m *ErrorDetails) String() string { return "ErrorDetails" }
func (*ErrorDetails) ProtoMessage()    {}
