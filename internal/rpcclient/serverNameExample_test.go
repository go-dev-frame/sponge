package rpcclient

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-dev-frame/sponge/pkg/utils"

	"github.com/go-dev-frame/sponge/configs"
	"github.com/go-dev-frame/sponge/internal/config"
)

func TestNewServerNameExampleRPCConn(t *testing.T) {
	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		t.Fatal(err)
	}
	config.Get().App.EnableTrace = true
	config.Get().App.EnableCircuitBreaker = true

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		config.Get().GrpcClient[0].RegistryDiscoveryType = "consul"
		NewServerNameExampleRPCConn()
		time.Sleep(time.Millisecond * 100)
		_ = CloseServerNameExampleRPCConn()
		cancel()
	})

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		config.Get().GrpcClient[0].RegistryDiscoveryType = "etcd"
		NewServerNameExampleRPCConn()
		time.Sleep(time.Millisecond * 100)
		_ = CloseServerNameExampleRPCConn()
		cancel()
	})

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		config.Get().GrpcClient[0].RegistryDiscoveryType = "nacos"
		NewServerNameExampleRPCConn()
		time.Sleep(time.Millisecond * 100)
		_ = CloseServerNameExampleRPCConn()
		cancel()
	})

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		config.Get().GrpcClient[0].Name = "unknown name"
		NewServerNameExampleRPCConn()
		cancel()
	})

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		conn := GetServerNameExampleRPCConn()
		assert.NotNil(t, conn)
		time.Sleep(time.Millisecond * 100)
		_ = CloseServerNameExampleRPCConn()
		cancel()
	})
}

func TestGetServerNameExampleRPCConn(t *testing.T) {
	serverNameExampleConn = nil
	err := CloseServerNameExampleRPCConn()
	assert.NoError(t, err)

	// nil error
	defer func() { recover() }()
	conn := GetServerNameExampleRPCConn()
	assert.NotNil(t, conn)
}
