package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"sync/atomic"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/base-org/blob-archiver/common/storage"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/log"
)

var ErrAlreadyStopped = errors.New("already stopped")

const (
	// 5 blocks per minute, 120 minutes
	twoHoursOfBlocks = 5 * 120
	// finalized l1 offset
	finalizedL1Offset = 64
	// Known log for any validation errors
	validationErrorLog = "validation error"
	// Number of attempts to fetch blobs from blob-api and beacon-node
	retryAttempts = 10
)

func NewValidator(l log.Logger, headerClient client.BeaconBlockHeadersProvider, beaconAPI BlobSidecarClient, blobAPI BlobSidecarClient, app context.CancelCauseFunc) *ValidatorService {
	return &ValidatorService{
		log:          l,
		headerClient: headerClient,
		beaconAPI:    beaconAPI,
		blobAPI:      blobAPI,
		closeApp:     app,
	}
}

type ValidatorService struct {
	stopped      atomic.Bool
	log          log.Logger
	headerClient client.BeaconBlockHeadersProvider
	beaconAPI    BlobSidecarClient
	blobAPI      BlobSidecarClient
	closeApp     context.CancelCauseFunc
}

// Start starts the validator service. This will fetch the current range of blocks to validate and start the validation
// process.
func (a *ValidatorService) Start(ctx context.Context) error {
	header, err := retry.Do(ctx, retryAttempts, retry.Exponential(), func() (*api.Response[*v1.BeaconBlockHeader], error) {
		return a.headerClient.BeaconBlockHeader(ctx, &api.BeaconBlockHeaderOpts{
			Block: "head",
		})
	})

	if err != nil {
		return fmt.Errorf("failed to get beacon block header: %w", err)
	}

	end := header.Data.Header.Message.Slot - finalizedL1Offset
	start := end - twoHoursOfBlocks

	go a.checkBlobs(ctx, start, end)

	return nil
}

// Stops the validator service.
func (a *ValidatorService) Stop(ctx context.Context) error {
	if a.stopped.Load() {
		return ErrAlreadyStopped
	}

	a.log.Info("Stopping validator")
	a.stopped.Store(true)

	return nil
}

func (a *ValidatorService) Stopped() bool {
	return a.stopped.Load()
}

// CheckBlobResult contains the summary of the blob checks
type CheckBlobResult struct {
	// ErrorFetching contains the list of slots for which the blob-api or beacon-node returned an error
	ErrorFetching []string
	// MismatchedStatus contains the list of slots for which the status code from the blob-api and beacon-node did not match
	MismatchedStatus []string
	// MismatchedData contains the list of slots for which the data from the blob-api and beacon-node did not match
	MismatchedData []string
}

// checkBlobs iterates all blocks in the range start:end and checks that the blobs from the beacon-node and blob-api
// are identical, when encoded in both JSON and SSZ.
func (a *ValidatorService) checkBlobs(ctx context.Context, start phase0.Slot, end phase0.Slot) CheckBlobResult {
	var result CheckBlobResult

	for slot := start; slot <= end; slot++ {
		for _, format := range []Format{FormatJson, FormatSSZ} {
			id := strconv.FormatUint(uint64(slot), 10)

			l := a.log.New("format", format, "slot", slot)

			blobStatus, blobResponse, blobError := retry.Do2(ctx, retryAttempts, retry.Exponential(), func() (int, storage.BlobSidecars, error) {
				return a.blobAPI.FetchSidecars(id, format)
			})

			if blobError != nil {
				result.ErrorFetching = append(result.ErrorFetching, id)
				l.Error(validationErrorLog, "reason", "error-blob-api", "error", blobError, "status", blobStatus)
				continue
			}

			beaconStatus, beaconResponse, beaconErr := retry.Do2(ctx, retryAttempts, retry.Exponential(), func() (int, storage.BlobSidecars, error) {
				return a.beaconAPI.FetchSidecars(id, format)
			})

			if beaconErr != nil {
				result.ErrorFetching = append(result.ErrorFetching, id)
				l.Error(validationErrorLog, "reason", "error-beacon-api", "error", beaconErr, "status", beaconStatus)
				continue
			}

			if beaconStatus != blobStatus {
				result.MismatchedStatus = append(result.MismatchedStatus, id)
				l.Error(validationErrorLog, "reason", "status-code-mismatch", "beaconStatus", beaconStatus, "blobStatus", blobStatus)
				continue
			}

			if beaconStatus != http.StatusOK {
				// This can happen if the slot has been missed
				l.Info("matching error status", "beacon", beaconStatus, "blob", blobStatus)
				continue

			}

			if !reflect.DeepEqual(beaconResponse, blobResponse) {
				result.MismatchedData = append(result.MismatchedData, id)
				l.Error(validationErrorLog, "reason", "response-mismatch")
			}

			l.Info("completed blob check", "blobs", len(beaconResponse.Data))
		}

		// Check if we should stop validation otherwise continue
		select {
		case <-ctx.Done():
			return result
		default:
			continue
		}
	}

	// Validation is complete, shutdown the app
	a.closeApp(nil)

	return result
}
