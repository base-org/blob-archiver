package metrics

import (
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type BlockSource string

var (
	MetricsNamespace = "blob_archiver"

	BlockSourceBackfill BlockSource = "backfill"
	BlockSourceLive     BlockSource = "live"
)

type Metricer interface {
	Registry() *prometheus.Registry
	RecordProcessedBlock(source BlockSource)
	RecordStoredBlobs(count int)
}

type metricsRecorder struct {
	blockProcessedCounter *prometheus.CounterVec
	blobsStored           prometheus.Counter
	registry              *prometheus.Registry
}

func NewMetrics() Metricer {
	registry := opmetrics.NewRegistry()
	factory := metrics.With(registry)
	return &metricsRecorder{
		registry: registry,
		blockProcessedCounter: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "blocks_processed",
			Help:      "number of times processing loop has run",
		}, []string{"source"}),
		blobsStored: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "blobs_stored",
			Help:      "number of blobs stored",
		}),
	}
}

func (m *metricsRecorder) Registry() *prometheus.Registry {
	return m.registry
}

func (m *metricsRecorder) RecordStoredBlobs(count int) {
	m.blobsStored.Add(float64(count))
}

func (m *metricsRecorder) RecordProcessedBlock(source BlockSource) {
	m.blockProcessedCounter.WithLabelValues(string(source)).Inc()
}
