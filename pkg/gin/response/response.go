// Package response provides wrapper gin returns json data in the same format.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/pkg/errcode"
)

// Result output data format
type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func newResp(code int, msg string, data interface{}) *Result {
	resp := &Result{
		Code: code,
		Msg:  msg,
	}

	// ensure that the data field is not nil on return, note that it is not nil when resp.data=[]interface {}, it is serialized to null
	if data == nil {
		resp.Data = &struct{}{}
	} else {
		resp.Data = data
	}

	return resp
}

func respJSONWithStatusCode(c *gin.Context, code int, msg string, data ...interface{}) {
	var firstData interface{}
	if len(data) > 0 {
		firstData = data[0]
	}
	resp := newResp(code, msg, firstData)
	c.JSON(code, resp)
}

// Output return standard HTTP status codes and message, parameter code is HTTP status code
func Output(c *gin.Context, code int, data ...interface{}) {
	switch code {
	case http.StatusOK:
		respJSONWithStatusCode(c, http.StatusOK, "ok", data...)
	case http.StatusBadRequest:
		respJSONWithStatusCode(c, http.StatusBadRequest, errcode.InvalidParams.Msg(), data...)
	case http.StatusUnauthorized:
		respJSONWithStatusCode(c, http.StatusUnauthorized, errcode.Unauthorized.Msg(), data...)
	case http.StatusForbidden:
		respJSONWithStatusCode(c, http.StatusForbidden, errcode.Forbidden.Msg(), data...)
	case http.StatusNotFound:
		respJSONWithStatusCode(c, http.StatusNotFound, errcode.NotFound.Msg(), data...)
	case http.StatusRequestTimeout:
		respJSONWithStatusCode(c, http.StatusRequestTimeout, errcode.Timeout.Msg(), data...)
	case http.StatusConflict:
		respJSONWithStatusCode(c, http.StatusConflict, errcode.Conflict.Msg(), data...)
	case http.StatusInternalServerError:
		respJSONWithStatusCode(c, http.StatusInternalServerError, errcode.InternalServerError.Msg(), data...)
	case http.StatusTooManyRequests:
		respJSONWithStatusCode(c, http.StatusTooManyRequests, errcode.LimitExceed.Msg(), data...)
	case http.StatusServiceUnavailable:
		respJSONWithStatusCode(c, http.StatusServiceUnavailable, errcode.ServiceUnavailable.Msg(), data...)

	default:
		respJSONWithStatusCode(c, code, http.StatusText(code), data...)
	}
}

// Out returns the standard HTTP status code and message, parameter err is errcode.Error
func Out(c *gin.Context, err *errcode.Error, data ...interface{}) {
	code := err.ToHTTPCode()
	switch code {
	case http.StatusOK:
		respJSONWithStatusCode(c, http.StatusOK, "ok", data...)
	case http.StatusInternalServerError:
		respJSONWithStatusCode(c, http.StatusInternalServerError, err.Msg(), data...)
	case http.StatusBadRequest:
		respJSONWithStatusCode(c, http.StatusBadRequest, err.Msg(), data...)
	case http.StatusUnauthorized:
		respJSONWithStatusCode(c, http.StatusUnauthorized, err.Msg(), data...)
	case http.StatusForbidden:
		respJSONWithStatusCode(c, http.StatusForbidden, err.Msg(), data...)
	case http.StatusNotFound:
		respJSONWithStatusCode(c, http.StatusNotFound, err.Msg(), data...)
	case http.StatusRequestTimeout:
		respJSONWithStatusCode(c, http.StatusRequestTimeout, err.Msg(), data...)
	case http.StatusConflict:
		respJSONWithStatusCode(c, http.StatusConflict, err.Msg(), data...)
	case http.StatusTooManyRequests:
		respJSONWithStatusCode(c, http.StatusTooManyRequests, err.Msg(), data...)
	case http.StatusServiceUnavailable:
		respJSONWithStatusCode(c, http.StatusServiceUnavailable, err.Msg(), data...)
	case http.StatusMethodNotAllowed:
		respJSONWithStatusCode(c, http.StatusMethodNotAllowed, err.Msg(), data...)

	default:
		respJSONWithStatusCode(c, http.StatusNotExtended, err.Msg(), data...)
	}
}

// status code flat 200, custom error codes in data.code
func respJSONWith200(c *gin.Context, code int, msg string, data ...interface{}) {
	var firstData interface{}
	if len(data) > 0 {
		firstData = data[0]
	}
	resp := newResp(code, msg, firstData)
	c.JSON(http.StatusOK, resp)
}

// Success return success
func Success(c *gin.Context, data ...interface{}) {
	respJSONWith200(c, 0, "ok", data...)
}

// Error return error
func Error(c *gin.Context, err *errcode.Error, data ...interface{}) {
	respJSONWith200(c, err.Code(), err.Msg(), data...)
}
