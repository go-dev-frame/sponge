package app

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type httpServer struct{}

func (h *httpServer) Start() error {
	fmt.Println("running http server")
	return nil
}

func (h *httpServer) Stop() error {
	return errors.New("mock stop http server")
}

func (h *httpServer) String() string {
	return ":8081"
}

type httpServer2 struct{}

func (h *httpServer2) Start() error {
	fmt.Println("running http server2")
	return nil
}

func (h *httpServer2) Stop() error {
	fmt.Println("stop http server")
	return nil
}

func (h *httpServer2) String() string {
	return ":8082"
}

type httpServer3 struct{}

func (h *httpServer3) Start() error {
	return errors.New("mock running http server3 error")
}

func (h *httpServer3) Stop() error {
	fmt.Println("stop http server3")
	return nil
}

func (h *httpServer3) String() string {
	return ":8083"
}

func TestApp(t *testing.T) {
	var (
		s       = &httpServer{}
		s2      = &httpServer2{}
		servers = []IServer{s, s2}

		closes = []Close{
			func() error {
				return s.Stop()
			},
			func() error {
				return s2.Stop()
			},
		}
	)

	a := New(servers, closes)
	go a.Run()
	time.Sleep(time.Second)

	// test watch
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	assert.Error(t, a.watch(ctx))

	time.Sleep(time.Second)
	t.Log(a.stop())
}

func TestAppError(t *testing.T) {
	s3 := &httpServer3{}
	servers := []IServer{s3}
	closes := []Close{
		func() error {
			return s3.Stop()
		},
	}

	a := New(servers, closes)
	go func() {
		defer func() {
			if e := recover(); e != nil {
				t.Log(e)
			}
		}()
		a.Run()
	}()
	time.Sleep(time.Second)
	t.Log(a.stop())
}
