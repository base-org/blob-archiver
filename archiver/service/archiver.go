package service

import (
	"context"
	"strconv"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/base-org/blob-archiver/archiver/flags"
	"github.com/base-org/blob-archiver/archiver/metrics"
	"github.com/base-org/blob-archiver/common/storage"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	liveFetchBlobMaximumRetries    = 10
	startupFetchBlobMaximumRetries = 3
	rearchiveMaximumRetries        = 3
	backfillErrorRetryInterval     = 5 * time.Second
)

func NewArchiver(l log.Logger, cfg flags.ArchiverConfig, dataStoreClient storage.DataStore, client BeaconClient, m metrics.Metricer) (*Archiver, error) {
	return &Archiver{
		log:             l,
		cfg:             cfg,
		dataStoreClient: dataStoreClient,
		metrics:         m,
		beaconClient:    client,
		stopCh:          make(chan struct{}),
	}, nil
}

type Archiver struct {
	log             log.Logger
	cfg             flags.ArchiverConfig
	dataStoreClient storage.DataStore
	beaconClient    BeaconClient
	metrics         metrics.Metricer
	stopCh          chan struct{}
}

// Start starts the archiver service. It begins polling the beacon node for the latest blocks and persisting blobs for
// them. Concurrently it'll also begin a backfill process (see backfillBlobs) to store all blobs from the current head
// to the previously stored blocks. This ensures that during restarts or outages of an archiver, any gaps will be
// filled in.
func (a *Archiver) Start(ctx context.Context) error {
	currentBlob, _, err := retry.Do2(ctx, startupFetchBlobMaximumRetries, retry.Exponential(), func() (*v1.BeaconBlockHeader, bool, error) {
		return a.persistBlobsForBlockToS3(ctx, "head", false)
	})

	if err != nil {
		a.log.Error("failed to seed archiver with initial block", "err", err)
		return err
	}

	go a.backfillBlobs(ctx, currentBlob)

	return a.trackLatestBlocks(ctx)
}

// Stops the archiver service.
func (a *Archiver) Stop(ctx context.Context) error {
	close(a.stopCh)
	return nil
}

// persistBlobsForBlockToS3 fetches the blobs for a given block and persists them to S3. It returns the block header
// and a boolean indicating whether the blobs already existed in S3 and any errors that occur.
// If the blobs are already stored, it will not overwrite the data. Currently, the archiver does not
// perform any validation of the blobs, it assumes a trusted beacon node. See:
// https://github.com/base-org/blob-archiver/issues/4.
func (a *Archiver) persistBlobsForBlockToS3(ctx context.Context, blockIdentifier string, overwrite bool) (*v1.BeaconBlockHeader, bool, error) {
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

	if exists && !overwrite {
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

	return currentHeader.Data, exists, nil
}

// backfillBlobs will persist all blobs from the provided beacon block header, to either the last block that was persisted
// to the archivers storage or the origin block in the configuration. This is used to ensure that any gaps can be filled.
// If an error is encountered persisting a block, it will retry after waiting for a period of time.
func (a *Archiver) backfillBlobs(ctx context.Context, latest *v1.BeaconBlockHeader) {
	current, alreadyExists, err := latest, false, error(nil)

	for !alreadyExists {
		if common.Hash(current.Root) == a.cfg.OriginBlock {
			a.log.Info("reached origin block", "hash", current.Root.String())
			return
		}

		previous := current
		current, alreadyExists, err = a.persistBlobsForBlockToS3(ctx, previous.Header.Message.ParentRoot.String(), false)
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
func (a *Archiver) trackLatestBlocks(ctx context.Context) error {
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
func (a *Archiver) processBlocksUntilKnownBlock(ctx context.Context) {
	a.log.Debug("refreshing live data")

	var start *v1.BeaconBlockHeader
	currentBlockId := "head"

	for {
		current, alreadyExisted, err := retry.Do2(ctx, liveFetchBlobMaximumRetries, retry.Exponential(), func() (*v1.BeaconBlockHeader, bool, error) {
			return a.persistBlobsForBlockToS3(ctx, currentBlockId, false)
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

func (a *Archiver) rearchiveRange(from uint64, to uint64) (uint64, uint64, error) {
	for i := from; i <= to; i++ {
		id := strconv.FormatUint(i, 10)

		a.log.Debug("rearchiving block", "blockId", id)

		_, _, err := retry.Do2(context.Background(), rearchiveMaximumRetries, retry.Exponential(), func() (*v1.BeaconBlockHeader, bool, error) {
			return a.persistBlobsForBlockToS3(context.Background(), id, true)
		})

		if err != nil {
			return from, i, err
		}

		a.metrics.RecordProcessedBlock(metrics.BlockSourceRearchive)
	}

	return from, to, nil
}
