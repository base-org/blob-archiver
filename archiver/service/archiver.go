package service

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/base-org/blob-archiver/archiver/flags"
	"github.com/base-org/blob-archiver/archiver/metrics"
	"github.com/base-org/blob-archiver/common/storage"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const liveFetchBlobMaximumRetries = 10
const startupFetchBlobMaximumRetries = 3
const backfillErrorRetryInterval = 5 * time.Second

var ErrAlreadyStopped = errors.New("already stopped")

type BeaconClient interface {
	client.BlobSidecarsProvider
	client.BeaconBlockHeadersProvider
}

func NewService(l log.Logger, cfg flags.ArchiverConfig, api *API, dataStoreClient storage.DataStore, client BeaconClient, m metrics.Metricer) (*ArchiverService, error) {
	return &ArchiverService{
		log:             l,
		cfg:             cfg,
		dataStoreClient: dataStoreClient,
		metrics:         m,
		stopCh:          make(chan struct{}),
		beaconClient:    client,
		api:             api,
	}, nil
}

type ArchiverService struct {
	stopped         atomic.Bool
	stopCh          chan struct{}
	log             log.Logger
	dataStoreClient storage.DataStore
	beaconClient    BeaconClient
	metricsServer   *httputil.HTTPServer
	cfg             flags.ArchiverConfig
	metrics         metrics.Metricer
	api             *API
}

// Start starts the archiver service. It begins polling the beacon node for the latest blocks and persisting blobs for
// them. Concurrently it'll also begin a backfill process (see backfillBlobs) to store all blobs from the current head
// to the previously stored blocks. This ensures that during restarts or outages of an archiver, any gaps will be
// filled in.
func (a *ArchiverService) Start(ctx context.Context) error {
	if a.cfg.MetricsConfig.Enabled {
		a.log.Info("starting metrics server", "addr", a.cfg.MetricsConfig.ListenAddr, "port", a.cfg.MetricsConfig.ListenPort)
		srv, err := opmetrics.StartServer(a.metrics.Registry(), a.cfg.MetricsConfig.ListenAddr, a.cfg.MetricsConfig.ListenPort)
		if err != nil {
			return err
		}

		a.log.Info("started metrics server", "addr", srv.Addr())
		a.metricsServer = srv
	}

	srv, err := httputil.StartHTTPServer(a.cfg.ListenAddr, a.api.router)
	if err != nil {
		return fmt.Errorf("failed to start Archiver API server: %w", err)
	}

	a.log.Info("Archiver API server started", "address", srv.Addr().String())

	currentBlob, _, err := retry.Do2(ctx, startupFetchBlobMaximumRetries, retry.Exponential(), func() (*v1.BeaconBlockHeader, bool, error) {
		return a.persistBlobsForBlockToS3(ctx, "head")
	})

	if err != nil {
		a.log.Error("failed to seed archiver with initial block", "err", err)
		return err
	}

	go a.backfillBlobs(ctx, currentBlob)

	return a.trackLatestBlocks(ctx)
}

// persistBlobsForBlockToS3 fetches the blobs for a given block and persists them to S3. It returns the block header
// and a boolean indicating whether the blobs already existed in S3 and any errors that occur.
// If the blobs are already stored, it will not overwrite the data. Currently, the archiver does not
// perform any validation of the blobs, it assumes a trusted beacon node. See:
// https://github.com/base-org/blob-archiver/issues/4.
func (a *ArchiverService) persistBlobsForBlockToS3(ctx context.Context, blockIdentifier string) (*v1.BeaconBlockHeader, bool, error) {
	currentHeader, err := a.beaconClient.BeaconBlockHeader(ctx, &api.BeaconBlockHeaderOpts{
		Block: blockIdentifier,
	})

	if err != nil {
		a.log.Error("failed to fetch latest beacon block header", "err", err)
		return nil, false, err
	}

	exists, err := a.dataStoreClient.Exists(ctx, common.Hash(currentHeader.Data.Root))
	if err != nil {
		a.log.Error("failed to check if blob exists", "err", err)
		return nil, false, err
	}

	if exists {
		a.log.Debug("blob already exists", "hash", currentHeader.Data.Root)
		return currentHeader.Data, true, nil
	}

	blobSidecars, err := a.beaconClient.BlobSidecars(ctx, &api.BlobSidecarsOpts{
		Block: currentHeader.Data.Root.String(),
	})

	if err != nil {
		a.log.Error("failed to fetch blob sidecars", "err", err)
		return nil, false, err
	}

	a.log.Debug("fetched blob sidecars", "count", len(blobSidecars.Data))

	blobData := storage.BlobData{
		Header: storage.Header{
			BeaconBlockHash: common.Hash(currentHeader.Data.Root),
		},
		BlobSidecars: storage.BlobSidecars{Data: blobSidecars.Data},
	}

	// The blob that is being written has not been validated. It is assumed that the beacon node is trusted.
	err = a.dataStoreClient.Write(ctx, blobData)

	if err != nil {
		a.log.Error("failed to write blob", "err", err)
		return nil, false, err
	}

	a.metrics.RecordStoredBlobs(len(blobSidecars.Data))

	return currentHeader.Data, false, nil
}

// Stops the archiver service.
func (a *ArchiverService) Stop(ctx context.Context) error {
	if a.stopped.Load() {
		return ErrAlreadyStopped
	}
	a.log.Info("Stopping Archiver")
	a.stopped.Store(true)

	close(a.stopCh)

	if a.metricsServer != nil {
		if err := a.metricsServer.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (a *ArchiverService) Stopped() bool {
	return a.stopped.Load()
}

// backfillBlobs will persist all blobs from the provided beacon block header, to either the last block that was persisted
// to the archivers storage or the origin block in the configuration. This is used to ensure that any gaps can be filled.
// If an error is encountered persisting a block, it will retry after waiting for a period of time.
func (a *ArchiverService) backfillBlobs(ctx context.Context, latest *v1.BeaconBlockHeader) {
	current, alreadyExists, err := latest, false, error(nil)

	for !alreadyExists {
		if common.Hash(current.Root) == a.cfg.OriginBlock {
			a.log.Info("reached origin block", "hash", current.Root.String())
			return
		}

		previous := current
		current, alreadyExists, err = a.persistBlobsForBlockToS3(ctx, previous.Header.Message.ParentRoot.String())
		if err != nil {
			a.log.Error("failed to persist blobs for block, will retry", "err", err, "hash", previous.Header.Message.ParentRoot.String())
			// Revert back to block we failed to fetch
			current = previous
			time.Sleep(backfillErrorRetryInterval)
			continue
		}

		if !alreadyExists {
			a.metrics.RecordProcessedBlock(metrics.BlockSourceBackfill)
		}
	}

	a.log.Info("backfill complete", "endHash", current.Root.String(), "startHash", latest.Root.String())
}

// trackLatestBlocks will poll the beacon node for the latest blocks and persist blobs for them.
func (a *ArchiverService) trackLatestBlocks(ctx context.Context) error {
	t := time.NewTicker(a.cfg.PollInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-a.stopCh:
			return nil
		case <-t.C:
			a.processBlocksUntilKnownBlock(ctx)
		}
	}
}

// processBlocksUntilKnownBlock will fetch and persist blobs for blocks until it finds a block that has been stored before.
// In the case of a reorg, it will fetch the new head and then walk back the chain, storing all blobs until it finds a
// known block -- that already exists in the archivers' storage.
func (a *ArchiverService) processBlocksUntilKnownBlock(ctx context.Context) {
	a.log.Debug("refreshing live data")

	var start *v1.BeaconBlockHeader
	currentBlockId := "head"

	for {
		current, alreadyExisted, err := retry.Do2(ctx, liveFetchBlobMaximumRetries, retry.Exponential(), func() (*v1.BeaconBlockHeader, bool, error) {
			return a.persistBlobsForBlockToS3(ctx, currentBlockId)
		})

		if err != nil {
			a.log.Error("failed to update live blobs for block", "err", err, "blockId", currentBlockId)
			return
		}

		if start == nil {
			start = current
		}

		if !alreadyExisted {
			a.metrics.RecordProcessedBlock(metrics.BlockSourceLive)
		} else {
			a.log.Debug("blob already exists", "hash", current.Root.String())
			break
		}

		currentBlockId = current.Header.Message.ParentRoot.String()
	}

	a.log.Info("live data refreshed", "startHash", start.Root.String(), "endHash", currentBlockId)
}
