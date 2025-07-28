package gotest

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/pkg/utils"
)

// Handler info
type Handler struct {
	TestData interface{}
	MockDao  *Dao
	IHandler interface{}

	Engine      *gin.Engine
	HTTPServer  *http.Server
	httpAddr    string
	requestAddr string
	routers     map[string]RouterInfo
}

// RouterInfo router info
type RouterInfo struct {
	FuncName    string
	Method      string
	Path        string
	HandlerFunc gin.HandlerFunc
}

// NewHandler instantiated handler
func NewHandler(dao *Dao, testData interface{}) *Handler {
	serverAddr, requestAddr := utils.GetLocalHTTPAddrPairs()

	return &Handler{
		TestData:    testData,
		MockDao:     dao,
		requestAddr: requestAddr,
		httpAddr:    serverAddr,
		routers:     make(map[string]RouterInfo),
	}
}

// GoRunHTTPServer run http server
func (h *Handler) GoRunHTTPServer(fns []RouterInfo) {
	if len(fns) == 0 {
		panic("HandlerFunc is empty")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	for _, fn := range fns {
		switch fn.Method {
		case http.MethodPost:
			r.POST(fn.Path, fn.HandlerFunc)
		case http.MethodDelete:
			r.DELETE(fn.Path, fn.HandlerFunc)
		case http.MethodPut:
			r.PUT(fn.Path, fn.HandlerFunc)
		case http.MethodPatch:
			r.PATCH(fn.Path, fn.HandlerFunc)
		case http.MethodGet:
			r.GET(fn.Path, fn.HandlerFunc)
		default:
			fmt.Println("unsupported http method " + fn.Method)
		}
		h.routers[strings.ToLower(fn.FuncName)] = fn
	}

	h.HTTPServer = &http.Server{
		Addr:    h.httpAddr,
		Handler: r,
	}

	go func() {
		if err := h.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
}

// GetRequestURL get request url from name
func (h *Handler) GetRequestURL(funcName string, pathVal ...interface{}) string {
	fn, ok := h.routers[strings.ToLower(funcName)]
	if !ok {
		return ""
	}

	varCount := strings.Count(fn.Path, "/:")
	if varCount == 0 || varCount != len(pathVal) {
		return h.requestAddr + "/" + strings.TrimLeft(fn.Path, "/")
	}

	ss := strings.Split(fn.Path, "/")
	var subPaths []string
	j := 0
	for _, s := range ss {
		if len(s) > 0 {
			if s[0] == ':' {
				subPaths = append(subPaths, fmt.Sprintf("%v", pathVal[j]))
				j++
			} else {
				subPaths = append(subPaths, s)
			}
		}
	}
	return h.requestAddr + "/" + strings.TrimLeft(strings.Join(subPaths, "/"), "/")
}

// Close handler
func (h *Handler) Close() {
	if h.MockDao != nil {
		h.MockDao.Close()
	}
	if h.HTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = h.HTTPServer.Shutdown(ctx)
	}
}
