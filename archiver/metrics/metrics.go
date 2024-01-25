package metrics

import (
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type BlockSource string

var (
	metricsNamespace = "blob_archiver"

	BlockSourceBackfill BlockSource = "backfill"
	BlockSourceLive     BlockSource = "live"
)

type Metricer interface {
	RecordProcessedBlock(source BlockSource)
	RecordStoredBlobs(count int)
}

type metricsRecorder struct {
	blockProcessedCounter *prometheus.CounterVec
	blobsStored           prometheus.Counter
}

func NewMetrics(registry *prometheus.Registry) Metricer {
	factory := metrics.With(registry)
	return &metricsRecorder{
		blockProcessedCounter: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "blocks_processed",
			Help:      "number of times processing loop has run",
		}, []string{"source"}),
		blobsStored: factory.NewCounter(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "blobs_stored",
			Help:      "number of blobs stored",
		}),
	}
}

func (m *metricsRecorder) RecordStoredBlobs(count int) {
	m.blobsStored.Add(float64(count))
}

func (m *metricsRecorder) RecordProcessedBlock(source BlockSource) {
	m.blockProcessedCounter.WithLabelValues(string(source)).Inc()
}
