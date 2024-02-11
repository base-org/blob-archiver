package service

import (
	"net/http"
	"time"

	m "github.com/base-org/blob-archiver/archiver/metrics"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	serverTimeout = 60 * time.Second
)

type API struct {
	router  *chi.Mux
	logger  log.Logger
	metrics m.Metricer
}

func NewAPI(metrics m.Metricer, logger log.Logger) *API {
	result := &API{
		router:  chi.NewRouter(),
		logger:  logger,
		metrics: metrics,
	}

	r := result.router
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(serverTimeout))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/healthz"))

	recorder := opmetrics.NewPromHTTPRecorder(metrics.Registry(), m.MetricsNamespace)
	r.Use(func(handler http.Handler) http.Handler {
		return opmetrics.NewHTTPRecordingMiddleware(recorder, handler)
	})

	r.Get("/", http.NotFound)

	return result
}
