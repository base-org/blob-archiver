package metrics

import (
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type BlockIdType string

var (
	MetricsNamespace = "blob_api"

	BlockIdTypeHash    BlockIdType = "hash"
	BlockIdTypeBeacon  BlockIdType = "beacon"
	BlockIdTypeInvalid BlockIdType = "invalid"
)

type Metricer interface {
	Registry() *prometheus.Registry
	RecordBlockIdType(t BlockIdType)
}

type metricsRecorder struct {
	inputType *prometheus.CounterVec
	registry  *prometheus.Registry
}

func NewMetrics() Metricer {
	registry := opmetrics.NewRegistry()
	factory := metrics.With(registry)
	return &metricsRecorder{
		registry: registry,
		inputType: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "block_id_type",
			Help:      "The type of block id used to request a block",
		}, []string{"type"}),
	}
}

func (m *metricsRecorder) RecordBlockIdType(t BlockIdType) {
	m.inputType.WithLabelValues(string(t)).Inc()
}

func (m *metricsRecorder) Registry() *prometheus.Registry {
	return m.registry
}
