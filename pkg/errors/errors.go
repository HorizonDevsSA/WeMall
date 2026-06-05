package errors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NotFound wraps a message in a gRPC NotFound status error.
func NotFound(msg string) error {
	return status.Error(codes.NotFound, msg)
}

// Unauthenticated wraps a message in a gRPC Unauthenticated status error.
func Unauthenticated(msg string) error {
	return status.Error(codes.Unauthenticated, msg)
}

// PermissionDenied wraps a message in a gRPC PermissionDenied status error.
func PermissionDenied(msg string) error {
	return status.Error(codes.PermissionDenied, msg)
}

// InvalidArgument wraps a message in a gRPC InvalidArgument status error.
func InvalidArgument(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}

// Internal wraps an error in a gRPC Internal status error.
func Internal(err error) error {
	_ = err
	return status.Error(codes.Internal, "internal server error")
}

// AlreadyExists wraps a message in a gRPC AlreadyExists status error.
func AlreadyExists(msg string) error {
	return status.Error(codes.AlreadyExists, msg)
}

// Unavailable wraps a message in a gRPC Unavailable status error.
func Unavailable(msg string) error {
	return status.Error(codes.Unavailable, msg)
}

// IsNotFound reports whether the error is a gRPC NotFound error.
func IsNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}
