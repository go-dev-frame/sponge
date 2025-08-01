package metrics

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/go-dev-frame/sponge/pkg/utils"
)

func Test_srvRegisterMetrics(t *testing.T) {
	opts := []Option{
		WithCounterMetrics(prometheus.NewCounterVec(prometheus.CounterOpts{Name: "demo1"}, []string{})),
		WithGaugeMetrics(prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "demo2"}, []string{})),
		WithHistogramMetrics(prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "demo3"}, []string{})),
		WithSummaryMetrics(prometheus.NewSummaryVec(prometheus.SummaryOpts{Name: "demo4"}, []string{})),
	}
	o := defaultMetricsOptions()
	o.apply(opts...)
	srvRegisterMetrics()
}

func TestWithCounterMetrics(t *testing.T) {
	testData := &prometheus.CounterVec{}
	opt := WithCounterMetrics(testData)
	o := new(options)
	o.apply(opt)
	assert.Contains(t, customizedCounterMetrics, testData)
}

func TestWithGaugeMetrics(t *testing.T) {
	testData := &prometheus.GaugeVec{}
	opt := WithGaugeMetrics(testData)
	o := new(options)
	o.apply(opt)
	assert.Contains(t, customizedGaugeMetrics, testData)
}

func TestWithHistogramMetrics(t *testing.T) {
	testData := &prometheus.HistogramVec{}
	opt := WithHistogramMetrics(testData)
	o := new(options)
	o.apply(opt)
	assert.Contains(t, customizedHistogramMetrics, testData)
}

func TestWithSummaryMetrics(t *testing.T) {
	testData := &prometheus.SummaryVec{}
	opt := WithSummaryMetrics(testData)
	o := new(options)
	o.apply(opt)
	assert.Contains(t, customizedSummaryMetrics, testData)
}

func Test_defaultMetricsOptions(t *testing.T) {
	o := defaultMetricsOptions()
	assert.NotNil(t, o)
}

func Test_metricsOptions_apply(t *testing.T) {
	testData := &prometheus.SummaryVec{}
	opt := WithSummaryMetrics(testData)
	o := defaultMetricsOptions()
	o.apply(opt)
	assert.Contains(t, customizedSummaryMetrics, testData)
}

func TestRegister(t *testing.T) {
	SetServerPattern("/rpc_server/metrics")
	Register(http.NewServeMux(), grpc.NewServer())
}

func TestServerHTTPService(t *testing.T) {
	serverAddr, _ := utils.GetLocalHTTPAddrPairs()
	s := ServerHTTPService(serverAddr, grpc.NewServer())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	time.Sleep(time.Millisecond * 100)
	err := s.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestStreamServerMetrics(t *testing.T) {
	metrics := StreamServerMetrics()
	assert.NotNil(t, metrics)
}

func TestUnaryServerMetrics(t *testing.T) {
	metrics := UnaryServerMetrics()
	assert.NotNil(t, metrics)
}
