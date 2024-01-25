package metrics

import (
	"github.com/ethereum-optimism/optimism/op-service/metrics"
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
	RecordBlockIdType(t BlockIdType)
}

type metricsRecorder struct {
	inputType *prometheus.CounterVec
}

func NewMetrics(registry *prometheus.Registry) Metricer {
	factory := metrics.With(registry)
	return &metricsRecorder{
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
