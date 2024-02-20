package blobtest

import (
	"crypto/rand"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	OriginBlock = common.Hash{9, 9, 9, 9, 9}
	One         = common.Hash{1}
	Two         = common.Hash{2}
	Three       = common.Hash{3}
	Four        = common.Hash{4}
	Five        = common.Hash{5}

	StartSlot = uint64(10)
	EndSlot   = uint64(15)
)

func RandBytes(t *testing.T, size uint) []byte {
	randomBytes := make([]byte, size)
	_, err := rand.Read(randomBytes)
	require.NoError(t, err)
	return randomBytes
}

func NewBlobSidecar(t *testing.T, i uint) *deneb.BlobSidecar {
	return &deneb.BlobSidecar{
		Index:         deneb.BlobIndex(i),
		Blob:          deneb.Blob(RandBytes(t, 131072)),
		KZGCommitment: deneb.KZGCommitment(RandBytes(t, 48)),
		KZGProof:      deneb.KZGProof(RandBytes(t, 48)),
		SignedBlockHeader: &phase0.SignedBeaconBlockHeader{
			Message: &phase0.BeaconBlockHeader{},
		},
	}
}

func NewBlobSidecars(t *testing.T, count uint) []*deneb.BlobSidecar {
	result := make([]*deneb.BlobSidecar, count)
	for i := uint(0); i < count; i++ {
		result[i] = NewBlobSidecar(t, i)
	}
	return result
}
