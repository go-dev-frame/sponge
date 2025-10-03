package response

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/go-dev-frame/sponge/pkg/errcode"
	"github.com/go-dev-frame/sponge/pkg/httpcli"
	"github.com/go-dev-frame/sponge/pkg/utils"
)

var (
	httpResponseCodes = []int{
		http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusRequestTimeout, http.StatusConflict, http.StatusInternalServerError,
	}
	outs = []*errcode.Error{
		errcode.Success, errcode.InvalidParams, errcode.Unauthorized, errcode.InternalServerError, errcode.NotFound,
		errcode.AlreadyExists, errcode.Timeout, errcode.TooManyRequests, errcode.Forbidden,
		errcode.MethodNotAllowed, errcode.ServiceUnavailable,
	}
)

func runResponseHTTPServer() string {
	serverAddr, requestAddr := utils.GetLocalHTTPAddrPairs()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/success", func(c *gin.Context) { Success(c, gin.H{"foo": "bar"}) })
	for _, out := range outs {
		if out.Code() == 0 {
			continue
		}
		r.GET(fmt.Sprintf("/error/code/%d", out.Code()), func(c *gin.Context) { Error(c, out) })
	}
	for _, code := range httpResponseCodes {
		r.GET(fmt.Sprintf("/code/%d", code), func(c *gin.Context) { Output(c, code) })
	}
	for _, out := range outs {
		r.GET(fmt.Sprintf("/out/code/%d", out.ToHTTPCode()), func(c *gin.Context) { Out(c, out) })
	}

	go func() {
		err := r.Run(serverAddr)
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Millisecond * 200)
	return requestAddr
}

func TestRespond(t *testing.T) {
	requestAddr := runResponseHTTPServer()

	result := &httpcli.StdResult{}
	err := httpcli.Get(result, requestAddr+"/success")
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Data)

	for _, out := range outs {
		if out.Code() == 0 {
			continue
		}
		result = &httpcli.StdResult{}
		url := fmt.Sprintf("%s/error/code/%d", requestAddr, out.Code())
		err = httpcli.Get(result, url)
		assert.NoError(t, err)
		assert.NotEqual(t, 0, result.Code)
	}

	for _, code := range httpResponseCodes {
		result = &httpcli.StdResult{}
		url := fmt.Sprintf("%s/code/%d", requestAddr, code)
		err = httpcli.Get(result, url)
		if code == http.StatusOK {
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, result.Code)
			continue
		}
		assert.Error(t, err)
	}
	for _, out := range outs {
		result = &httpcli.StdResult{}
		url := fmt.Sprintf("%s/out/code/%d", requestAddr, out.ToHTTPCode())
		err = httpcli.Get(result, url)
		if out.ToHTTPCode() == http.StatusOK {
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, result.Code)
			continue
		}
		assert.Error(t, err)
	}
}
