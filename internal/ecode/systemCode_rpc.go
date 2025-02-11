package ecode

import (
	"github.com/go-dev-frame/sponge/pkg/errcode"
)

// rpc system level error code, with status prefix, error code range 30000~40000
var (
	StatusSuccess = errcode.StatusSuccess

	StatusCanceled            = errcode.StatusCanceled
	StatusUnknown             = errcode.StatusUnknown
	StatusInvalidParams       = errcode.StatusInvalidParams
	StatusDeadlineExceeded    = errcode.StatusDeadlineExceeded
	StatusNotFound            = errcode.StatusNotFound
	StatusAlreadyExists       = errcode.StatusAlreadyExists
	StatusPermissionDenied    = errcode.StatusPermissionDenied
	StatusResourceExhausted   = errcode.StatusResourceExhausted
	StatusFailedPrecondition  = errcode.StatusFailedPrecondition
	StatusAborted             = errcode.StatusAborted
	StatusOutOfRange          = errcode.StatusOutOfRange
	StatusUnimplemented       = errcode.StatusUnimplemented
	StatusInternalServerError = errcode.StatusInternalServerError
	StatusServiceUnavailable  = errcode.StatusServiceUnavailable
	StatusDataLoss            = errcode.StatusDataLoss
	StatusUnauthorized        = errcode.StatusUnauthorized

	StatusTimeout          = errcode.StatusTimeout
	StatusTooManyRequests  = errcode.StatusTooManyRequests
	StatusForbidden        = errcode.StatusForbidden
	StatusLimitExceed      = errcode.StatusLimitExceed
	StatusMethodNotAllowed = errcode.StatusMethodNotAllowed
	StatusAccessDenied     = errcode.StatusAccessDenied
	StatusConflict         = errcode.StatusConflict
)

// Any kev-value
func Any(key string, val interface{}) errcode.Detail {
	return errcode.Any(key, val)
}

// StatusSkipResponse is only use for grpc-gateway
var StatusSkipResponse = errcode.SkipResponse

// GetStatusCode get status code from error returned by RPC invoke
var GetStatusCode = errcode.GetStatusCode
