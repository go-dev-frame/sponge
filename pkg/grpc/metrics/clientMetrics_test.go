package metrics

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-dev-frame/sponge/pkg/utils"
)

func TestClientHTTPService(t *testing.T) {
	serverAddr, _ := utils.GetLocalHTTPAddrPairs()

	s := ClientHTTPService(serverAddr)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	time.Sleep(time.Millisecond * 100)
	err := s.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestStreamClientMetrics(t *testing.T) {
	metrics := StreamClientMetrics()
	assert.NotNil(t, metrics)
}

func TestUnaryClientMetrics(t *testing.T) {
	metrics := UnaryClientMetrics()
	assert.NotNil(t, metrics)
}

func Test_cliRegisterMetrics(t *testing.T) {
	cliRegisterMetrics()
}

func TestClientRegister(t *testing.T) {
	SetClientPattern("/rpc_client/metrics")
	ClientRegister(http.NewServeMux())
}
