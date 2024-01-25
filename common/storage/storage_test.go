package storage

import (
	"testing"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/base-org/blob-archiver/common/blobtest"
	"github.com/stretchr/testify/require"
)

func TestMarshalSSZ(t *testing.T) {
	b := &BlobSidecars{
		Data: []*deneb.BlobSidecar{
			{
				Index:         1,
				Blob:          deneb.Blob(blobtest.RandBytes(t, 131072)),
				KZGCommitment: deneb.KZGCommitment(blobtest.RandBytes(t, 48)),
				KZGProof:      deneb.KZGProof(blobtest.RandBytes(t, 48)),
			},
			{
				Index:         2,
				Blob:          deneb.Blob(blobtest.RandBytes(t, 131072)),
				KZGCommitment: deneb.KZGCommitment(blobtest.RandBytes(t, 48)),
				KZGProof:      deneb.KZGProof(blobtest.RandBytes(t, 48)),
			},
		},
	}

	data, err := b.MarshalSSZ()
	require.NoError(t, err)

	sidecars := api.BlobSidecars{}
	err = sidecars.UnmarshalSSZ(data)
	require.NoError(t, err)

	require.Equal(t, len(b.Data), len(sidecars.Sidecars))
	for i := range b.Data {
		require.Equal(t, b.Data[i].Index, sidecars.Sidecars[i].Index)
		require.Equal(t, b.Data[i].Blob, sidecars.Sidecars[i].Blob)
		require.Equal(t, b.Data[i].KZGCommitment, sidecars.Sidecars[i].KZGCommitment)
		require.Equal(t, b.Data[i].KZGProof, sidecars.Sidecars[i].KZGProof)
	}
}
