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
	ErrNotFound = errors.New("blob not found")
	ErrStorage  = errors.New("error accessing storage")
	ErrEncoding = errors.New("error encoding/decoding blob")
)

type Header struct {
	BeaconBlockHash common.Hash `json:"beacon_block_hash"`
}

type BlobSidecars struct {
	Data []*deneb.BlobSidecar `json:"data"`
}

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

type DataStoreReader interface {
	Exists(ctx context.Context, hash common.Hash) (bool, error)
	Read(ctx context.Context, hash common.Hash) (BlobData, error)
}

type DataStoreWriter interface {
	Write(ctx context.Context, data BlobData) error
}

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
