package apperr

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AppError struct {
	inner    *fiber.Error
	grpcCode codes.Code
}

func ToGRPCError(err error) error {
	return From(err).ToGRPCError()
}

func New(code int, message ...string) *AppError {
	err := &AppError{
		inner: &fiber.Error{
			Code:    code,
			Message: utils.StatusMessage(code),
		},
		grpcCode: httpToGRPCCode(code),
	}
	if len(message) > 0 {
		err.inner.Message = message[0]
	}
	return err
}

func From(err error) *AppError {
	if err == nil {
		return nil
	}

	// Если уже AppError, возвращаем как есть
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	// Если fiber.Error, берём его код
	if fiberErr, ok := err.(*fiber.Error); ok {
		return &AppError{
			inner:    fiberErr,
			grpcCode: httpToGRPCCode(fiberErr.Code),
		}
	}

	// Если gRPC status.Error
	if st, ok := status.FromError(err); ok {
		httpCode := grpcToHTTPCode(st.Code())
		return &AppError{
			inner: &fiber.Error{
				Code:    httpCode,
				Message: st.Message(),
			},
			grpcCode: st.Code(),
		}
	}

	// Для других ошибок — внутренняя ошибка сервера
	return &AppError{
		inner: &fiber.Error{
			Code:    fiber.StatusInternalServerError,
			Message: err.Error(),
		},
		grpcCode: codes.Internal,
	}
}

func (err *AppError) WithMessage(message string) *AppError {
	err.inner.Message = message
	return err
}

func (err *AppError) Code() int {
	return err.inner.Code
}

func (err *AppError) GRPCCode() codes.Code {
	return err.grpcCode
}

func (err *AppError) Message() string {
	return err.inner.Message
}

func (err *AppError) Error() string {
	return err.inner.Error()
}

func (err *AppError) ToGRPCError() error {
	return status.Error(err.grpcCode, err.Message())
}

// Errors
var (
	ErrBadRequest                   = From(fiber.ErrBadRequest)                   // 400
	ErrUnauthorized                 = From(fiber.ErrUnauthorized)                 // 401
	ErrPaymentRequired              = From(fiber.ErrPaymentRequired)              // 402
	ErrForbidden                    = From(fiber.ErrForbidden)                    // 403
	ErrNotFound                     = From(fiber.ErrNotFound)                     // 404
	ErrMethodNotAllowed             = From(fiber.ErrMethodNotAllowed)             // 405
	ErrNotAcceptable                = From(fiber.ErrNotAcceptable)                // 406
	ErrProxyAuthRequired            = From(fiber.ErrProxyAuthRequired)            // 407
	ErrRequestTimeout               = From(fiber.ErrRequestTimeout)               // 408
	ErrConflict                     = From(fiber.ErrConflict)                     // 409
	ErrGone                         = From(fiber.ErrGone)                         // 410
	ErrLengthRequired               = From(fiber.ErrLengthRequired)               // 411
	ErrPreconditionFailed           = From(fiber.ErrPreconditionFailed)           // 412
	ErrRequestEntityTooLarge        = From(fiber.ErrRequestEntityTooLarge)        // 413
	ErrRequestURITooLong            = From(fiber.ErrRequestURITooLong)            // 414
	ErrUnsupportedMediaType         = From(fiber.ErrUnsupportedMediaType)         // 415
	ErrRequestedRangeNotSatisfiable = From(fiber.ErrRequestedRangeNotSatisfiable) // 416
	ErrExpectationFailed            = From(fiber.ErrExpectationFailed)            // 417
	ErrTeapot                       = From(fiber.ErrTeapot)                       // 418
	ErrMisdirectedRequest           = From(fiber.ErrMisdirectedRequest)           // 421
	ErrUnprocessableEntity          = From(fiber.ErrUnprocessableEntity)          // 422
	ErrLocked                       = From(fiber.ErrLocked)                       // 423
	ErrFailedDependency             = From(fiber.ErrFailedDependency)             // 424
	ErrTooEarly                     = From(fiber.ErrTooEarly)                     // 425
	ErrUpgradeRequired              = From(fiber.ErrUpgradeRequired)              // 426
	ErrPreconditionRequired         = From(fiber.ErrPreconditionRequired)         // 428
	ErrTooManyRequests              = From(fiber.ErrTooManyRequests)              // 429
	ErrRequestHeaderFieldsTooLarge  = From(fiber.ErrRequestHeaderFieldsTooLarge)  // 431
	ErrUnavailableForLegalReasons   = From(fiber.ErrUnavailableForLegalReasons)   // 451

	ErrInternalServerError           = From(fiber.ErrInternalServerError)           // 500
	ErrNotImplemented                = From(fiber.ErrNotImplemented)                // 501
	ErrBadGateway                    = From(fiber.ErrBadGateway)                    // 502
	ErrServiceUnavailable            = From(fiber.ErrServiceUnavailable)            // 503
	ErrGatewayTimeout                = From(fiber.ErrGatewayTimeout)                // 504
	ErrHTTPVersionNotSupported       = From(fiber.ErrHTTPVersionNotSupported)       // 505
	ErrVariantAlsoNegotiates         = From(fiber.ErrVariantAlsoNegotiates)         // 506
	ErrInsufficientStorage           = From(fiber.ErrInsufficientStorage)           // 507
	ErrLoopDetected                  = From(fiber.ErrLoopDetected)                  // 508
	ErrNotExtended                   = From(fiber.ErrNotExtended)                   // 510
	ErrNetworkAuthenticationRequired = From(fiber.ErrNetworkAuthenticationRequired) // 511
)

func httpToGRPCCode(httpCode int) codes.Code {
	switch httpCode {
	case fiber.StatusBadRequest:
		return codes.InvalidArgument
	case fiber.StatusUnauthorized:
		return codes.Unauthenticated
	case fiber.StatusForbidden:
		return codes.PermissionDenied
	case fiber.StatusNotFound:
		return codes.NotFound
	case fiber.StatusConflict:
		return codes.AlreadyExists
	case fiber.StatusTooManyRequests:
		return codes.ResourceExhausted
	case fiber.StatusRequestTimeout:
		return codes.DeadlineExceeded
	case fiber.StatusNotImplemented:
		return codes.Unimplemented
	case fiber.StatusServiceUnavailable:
		return codes.Unavailable
	case fiber.StatusInternalServerError, fiber.StatusBadGateway, fiber.StatusGatewayTimeout:
		return codes.Internal
	default:
		return codes.Unknown
	}
}

func grpcToHTTPCode(grpcCode codes.Code) int {
	switch grpcCode {
	case codes.InvalidArgument, codes.OutOfRange:
		return fiber.StatusBadRequest
	case codes.Unauthenticated:
		return fiber.StatusUnauthorized
	case codes.PermissionDenied:
		return fiber.StatusForbidden
	case codes.NotFound:
		return fiber.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return fiber.StatusConflict
	case codes.FailedPrecondition:
		return fiber.StatusPreconditionFailed
	case codes.ResourceExhausted:
		return fiber.StatusTooManyRequests
	case codes.Canceled, codes.DeadlineExceeded:
		return fiber.StatusRequestTimeout
	case codes.Unimplemented:
		return fiber.StatusNotImplemented
	case codes.Unavailable:
		return fiber.StatusServiceUnavailable
	case codes.Internal, codes.Unknown, codes.DataLoss:
		return fiber.StatusInternalServerError
	default:
		return fiber.StatusInternalServerError
	}
}
