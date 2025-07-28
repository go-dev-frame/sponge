package gotest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newHandler() *Handler {
	var testData = map[string]interface{}{
		"1": "foo",
		"2": "bar",
	}

	// init mock cache
	c := NewCache(map[string]interface{}{"no cache": testData})
	c.ICache = struct{}{} // instantiated cache interface

	// init mock dao
	d := NewDao(c, testData)
	d.IDao = struct{}{} // instantiated dao interface

	// init mock handler
	h := NewHandler(d, testData)
	h.IHandler = struct{}{} // instantiated handler interface

	return h
}

func TestNewHandler(t *testing.T) {
	h := newHandler()
	defer h.Close()
}

func TestHandler_GetRequestURL(t *testing.T) {
	h := newHandler()
	defer h.Close()

	h.GetRequestURL("/path")
}

func TestHandler_GoRunHttpServer(t *testing.T) {
	h := newHandler()
	defer h.Close()

	handlerFunc := func(c *gin.Context) {
		c.String(http.StatusOK, "hello world!")
	}

	h.GoRunHTTPServer([]RouterInfo{
		{
			FuncName:    "create",
			Method:      http.MethodPost,
			Path:        "/user",
			HandlerFunc: handlerFunc,
		},
		{
			FuncName:    "deleteByID",
			Method:      http.MethodDelete,
			Path:        "/user/:id",
			HandlerFunc: handlerFunc,
		},
		{
			FuncName:    "updateByID",
			Method:      http.MethodPut,
			Path:        "/user/:id",
			HandlerFunc: handlerFunc,
		},
		{
			FuncName:    "updateByID2",
			Method:      http.MethodPatch,
			Path:        "/user2/:id",
			HandlerFunc: handlerFunc,
		},
		{
			FuncName:    "getById",
			Method:      http.MethodGet,
			Path:        "/user/:id",
			HandlerFunc: handlerFunc,
		},
		{
			FuncName:    "options",
			Method:      http.MethodOptions,
			Path:        "/user",
			HandlerFunc: handlerFunc,
		},
	})

	time.Sleep(time.Millisecond * 200)
	url := h.GetRequestURL("updateByID", 1)
	t.Log(url)

	time.Sleep(time.Millisecond * 100)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = h.HTTPServer.Shutdown(ctx)
}
