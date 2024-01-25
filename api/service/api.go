package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	m "github.com/base-org/blob-archiver/api/metrics"
	"github.com/base-org/blob-archiver/common/storage"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

type httpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e httpError) write(w http.ResponseWriter) {
	w.WriteHeader(e.Code)
	_ = json.NewEncoder(w).Encode(e)
}

func (e httpError) Error() string {
	return e.Message
}

const (
	sszAcceptType = "application/octet-stream"
	serverTimeout = 60 * time.Second
)

var (
	errUnknownBlock = &httpError{
		Code:    http.StatusNotFound,
		Message: "Block not found",
	}
	errServerError = &httpError{
		Code:    http.StatusInternalServerError,
		Message: "Internal server error",
	}
)

func newBlockIdError(input string) *httpError {
	return &httpError{
		Code:    http.StatusBadRequest,
		Message: fmt.Sprintf("invalid block id: %s", input),
	}
}

func newIndicesError(input string) *httpError {
	return &httpError{
		Code:    http.StatusBadRequest,
		Message: fmt.Sprintf("invalid index input: %s", input),
	}
}

type API struct {
	dataStoreClient storage.DataStoreReader
	beaconClient    client.BeaconBlockHeadersProvider
	router          *chi.Mux
	logger          log.Logger
	metrics         m.Metricer
}

func NewAPI(dataStoreClient storage.DataStoreReader, beaconClient client.BeaconBlockHeadersProvider, metrics m.Metricer, registry *prometheus.Registry, logger log.Logger) *API {
	result := &API{
		dataStoreClient: dataStoreClient,
		beaconClient:    beaconClient,
		router:          chi.NewRouter(),
		logger:          logger,
		metrics:         metrics,
	}

	r := result.router
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(serverTimeout))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/healthz"))

	recorder := opmetrics.NewPromHTTPRecorder(registry, m.MetricsNamespace)
	r.Use(func(handler http.Handler) http.Handler {
		return opmetrics.NewHTTPRecordingMiddleware(recorder, handler)
	})

	r.Get("/eth/v1/beacon/blob_sidecars/{id}", result.blobSidecarHandler)

	return result
}

func isHash(s string) bool {
	if len(s) != 66 || !strings.HasPrefix(s, "0x") {
		return false
	}

	_, err := hexutil.Decode(s)
	return err == nil
}

func isSlot(id string) bool {
	_, err := strconv.ParseUint(id, 10, 64)
	return err == nil
}

func isKnownIdentifier(id string) bool {
	return slices.Contains([]string{"genesis", "finalized", "head"}, id)
}

// toBeaconBlockHash converts a string that can be a slot, hash or identifier to a beacon block hash.
func (a *API) toBeaconBlockHash(id string) (common.Hash, *httpError) {
	if isHash(id) {
		a.metrics.RecordBlockIdType(m.BlockIdTypeHash)
		return common.HexToHash(id), nil
	} else if isSlot(id) || isKnownIdentifier(id) {
		a.metrics.RecordBlockIdType(m.BlockIdTypeBeacon)
		result, err := a.beaconClient.BeaconBlockHeader(context.Background(), &api.BeaconBlockHeaderOpts{
			Common: api.CommonOpts{},
			Block:  id,
		})

		if err != nil {
			var apiErr *api.Error
			if errors.As(err, &apiErr) {
				switch apiErr.StatusCode {
				case 404:
					return common.Hash{}, errUnknownBlock
				}
			}

			return common.Hash{}, errServerError
		}

		return common.Hash(result.Data.Root), nil
	} else {
		a.metrics.RecordBlockIdType(m.BlockIdTypeInvalid)
		return common.Hash{}, newBlockIdError(id)
	}
}

func (a *API) blobSidecarHandler(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "id")
	beaconBlockHash, err := a.toBeaconBlockHash(param)
	if err != nil {
		err.write(w)
		return
	}

	result, storageErr := a.dataStoreClient.Read(r.Context(), beaconBlockHash)
	if storageErr != nil {
		if errors.Is(storageErr, storage.ErrNotFound) {
			errUnknownBlock.write(w)
		} else {
			a.logger.Info("unexpected error fetching blobs", "err", storageErr, "beaconBlockHash", beaconBlockHash.String(), "param", param)
			errServerError.write(w)
		}
		return
	}

	blobSidecars := result.BlobSidecars

	filteredBlobSidecars, err := filterBlobs(blobSidecars.Data, r.URL.Query().Get("indices"))
	if err != nil {
		err.write(w)
		return
	}

	blobSidecars.Data = filteredBlobSidecars
	responseType := r.Header.Get("Accept")

	if responseType == sszAcceptType {
		res, err := blobSidecars.MarshalSSZ()
		if err != nil {
			a.logger.Error("unable to marshal blob sidecars to SSZ", "err", err)
			errServerError.write(w)
			return
		}

		_, err = w.Write(res)

		if err != nil {
			a.logger.Error("unable to write ssz response", "err", err)
			errServerError.write(w)
			return
		}
	} else {
		err := json.NewEncoder(w).Encode(blobSidecars)
		if err != nil {
			a.logger.Error("unable to encode blob sidecars to JSON", "err", err)
			errServerError.write(w)
			return
		}
	}
}

// filterBlobs filters the blobs based on the indices query provided.
// If no indices or invalid indices are provided, the original blobs are returned.
func filterBlobs(blobs []*deneb.BlobSidecar, indices string) ([]*deneb.BlobSidecar, *httpError) {
	if indices == "" {
		return blobs, nil
	}

	splits := strings.Split(indices, ",")
	if len(splits) == 0 {
		return blobs, nil
	}

	indicesMap := map[deneb.BlobIndex]bool{}
	for _, index := range splits {
		parsedInt, err := strconv.ParseUint(index, 10, 64)
		if err != nil {
			return nil, newIndicesError(index)
		}
		blobIndex := deneb.BlobIndex(parsedInt)
		indicesMap[blobIndex] = true
	}

	filteredBlobs := make([]*deneb.BlobSidecar, 0)
	for _, blob := range blobs {
		if _, ok := indicesMap[blob.Index]; ok {
			filteredBlobs = append(filteredBlobs, blob)
		}
	}

	return filteredBlobs, nil
}
