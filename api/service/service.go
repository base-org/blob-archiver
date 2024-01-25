package service

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	client "github.com/attestantio/go-eth2-client"
	"github.com/base-org/blob-archiver/api/flags"
	"github.com/base-org/blob-archiver/api/metrics"
	"github.com/base-org/blob-archiver/common/storage"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

var ErrAlreadyStopped = errors.New("already stopped")

func NewAPIService(l log.Logger, dataStoreClient storage.DataStoreReader, beaconClient client.BeaconBlockHeadersProvider, cfg flags.APIConfig, registry *prometheus.Registry, m metrics.Metricer) *APIService {
	router := NewAPI(dataStoreClient, beaconClient, m, registry, l)
	return &APIService{
		log:      l,
		cfg:      cfg,
		registry: registry,
		router:   router,
	}
}

type APIService struct {
	stopped       atomic.Bool
	log           log.Logger
	cfg           flags.APIConfig
	registry      *prometheus.Registry
	metricsServer *httputil.HTTPServer
	apiServer     *httputil.HTTPServer
	router        *API
}

func (a *APIService) Start(ctx context.Context) error {
	if a.cfg.MetricsConfig.Enabled {
		a.log.Info("starting metrics server", "addr", a.cfg.MetricsConfig.ListenAddr, "port", a.cfg.MetricsConfig.ListenPort)
		srv, err := opmetrics.StartServer(a.registry, a.cfg.MetricsConfig.ListenAddr, a.cfg.MetricsConfig.ListenPort)
		if err != nil {
			return err
		}

		a.log.Info("started metrics server", "addr", srv.Addr())
		a.metricsServer = srv
	}

	a.log.Debug("starting API server", "address", a.cfg.ListenAddr)

	srv, err := httputil.StartHTTPServer(a.cfg.ListenAddr, a.router.router)
	if err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	a.log.Info("API server started", "address", srv.Addr().String())
	a.apiServer = srv
	return nil
}

func (a *APIService) Stop(ctx context.Context) error {
	if a.stopped.Load() {
		return ErrAlreadyStopped
	}
	a.log.Info("Stopping Archiver")
	a.stopped.Store(true)

	if a.apiServer != nil {
		if err := a.apiServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	if a.metricsServer != nil {
		if err := a.metricsServer.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (a *APIService) Stopped() bool {
	return a.stopped.Load()
}
