package gqlerrors

import (
	"context"
	"errors"
	"log"
	"runtime/debug"
	"strings"

	"github.com/vektah/gqlparser/v2/gqlerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	codeUnauthenticated = "UNAUTHENTICATED"
	codeForbidden       = "FORBIDDEN"
	codeBadRequest      = "BAD_REQUEST"
	codeNotFound        = "NOT_FOUND"
	codeConflict        = "CONFLICT"
	codeServiceDown     = "SERVICE_UNAVAILABLE"
	codeInternal        = "INTERNAL_SERVER_ERROR"
)

func Unauthenticated(msg string) error {
	return errors.New(codeUnauthenticated + ": " + msg)
}

func Forbidden(msg string) error {
	return errors.New(codeForbidden + ": " + msg)
}

// ErrorPresenter converts any error into a gqlerror.Error with a code extension.
// Signature matches graphql.ErrorPresenterFunc.
func ErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	_ = ctx
	log.Printf("GRAPHQL ERROR: %v (%T)", err, err)
	code, message := classify(err)
	ge := &gqlerror.Error{
		Message:    message,
		Extensions: map[string]interface{}{"code": code},
	}

	// Keep mapped transport code for diagnostics without leaking internals.
	if st, ok := status.FromError(err); ok {
		ge.Extensions["grpc_code"] = st.Code().String()
	}
	return ge
}

// RecoverFunc handles panics inside resolvers.
// Signature matches graphql.RecoverFunc.
func RecoverFunc(ctx context.Context, rec interface{}) error {
	_ = ctx
	log.Printf("PANIC RECOVERED: %v\n%s", rec, debug.Stack())
	return errors.New(codeInternal + ": internal server error")
}

func classify(err error) (string, string) {
	if err == nil {
		return codeInternal, "internal server error"
	}

	msg := err.Error()
	if strings.HasPrefix(msg, codeUnauthenticated+":") {
		return codeUnauthenticated, strings.TrimSpace(strings.TrimPrefix(msg, codeUnauthenticated+":"))
	}
	if strings.HasPrefix(msg, codeForbidden+":") {
		return codeForbidden, strings.TrimSpace(strings.TrimPrefix(msg, codeForbidden+":"))
	}

	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unauthenticated:
			return codeUnauthenticated, st.Message()
		case codes.PermissionDenied:
			return codeForbidden, st.Message()
		case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
			return codeBadRequest, st.Message()
		case codes.NotFound:
			return codeNotFound, st.Message()
		case codes.AlreadyExists, codes.Aborted:
			return codeConflict, st.Message()
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Canceled:
			return codeServiceDown, "service temporarily unavailable"
		default:
			return codeInternal, "internal server error"
		}
	}

	return codeInternal, "internal server error"
}
