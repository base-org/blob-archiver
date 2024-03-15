package storage

import (
	"context"
	"errors"

	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/base-org/blob-archiver/common/flags"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	blobSidecarSize = 131928
)

var (
	// ErrNotFound is returned when a blob is not found in the storage.
	ErrNotFound = errors.New("blob not found")
	// ErrStorage is returned when there is an error accessing the storage.
	ErrStorage = errors.New("error accessing storage")
	// ErrMarshaling is returned when there is an error in (un)marshaling the blob
	ErrMarshaling = errors.New("error encoding/decoding blob")
	// ErrCompress is returned when there is an error gzipping the data
	ErrCompress = errors.New("error compressing blob")
)

type Header struct {
	BeaconBlockHash common.Hash `json:"beacon_block_hash"`
}

type BlobSidecars struct {
	Data []*deneb.BlobSidecar `json:"data"`
}

// MarshalSSZ marshals the blob sidecars into SSZ. As the blob sidecars are a single list of fixed size elements, we can
// simply concatenate the marshaled sidecars together.
func (b *BlobSidecars) MarshalSSZ() ([]byte, error) {
	result := make([]byte, b.SizeSSZ())

	for i, sidecar := range b.Data {
		sidecarBytes, err := sidecar.MarshalSSZ()
		if err != nil {
			return nil, err
		}

		from := i * len(sidecarBytes)
		to := (i + 1) * len(sidecarBytes)

		copy(result[from:to], sidecarBytes)
	}

	return result, nil
}

func (b *BlobSidecars) SizeSSZ() int {
	return len(b.Data) * blobSidecarSize
}

type BlobData struct {
	Header       Header       `json:"header"`
	BlobSidecars BlobSidecars `json:"blob_sidecars"`
}

// DataStoreReader is the interface for reading from a data store.
type DataStoreReader interface {
	// Exists returns true if the given blob hash exists in the data store, false otherwise.
	// It should return one of the following:
	// - nil: the existence check was successful. In this case the boolean should also be set correctly.
	// - ErrStorage: there was an error accessing the data store.
	Exists(ctx context.Context, hash common.Hash) (bool, error)
	// Read reads the blob data for the given beacon block hash from the data store.
	// It should return one of the following:
	// - nil: reading the blob was successful. The blob data is also returned.
	// - ErrNotFound: the blob data was not found in the data store.
	// - ErrStorage: there was an error accessing the data store.
	// - ErrMarshaling: there was an error decoding the blob data.
	Read(ctx context.Context, hash common.Hash) (BlobData, error)
}

// DataStoreWriter is the interface for writing to a data store.
type DataStoreWriter interface {
	// Write writes the given blob data to the data store. It should return one of the following errors:
	// - nil: writing the blob was successful.
	// - ErrStorage: there was an error accessing the data store.
	// - ErrMarshaling: there was an error encoding the blob data.
	Write(ctx context.Context, data BlobData) error
}

// DataStore is the interface for a data store that can be both written to and read from.
type DataStore interface {
	DataStoreReader
	DataStoreWriter
}

func NewStorage(cfg flags.StorageConfig, l log.Logger) (DataStore, error) {
	if cfg.DataStorageType == flags.DataStorageS3 {
		return NewS3Storage(cfg.S3Config, l)
	} else {
		return NewFileStorage(cfg.FileStorageDirectory, l), nil
	}
}
