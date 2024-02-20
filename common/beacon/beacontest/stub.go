package beacontest

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/base-org/blob-archiver/common/blobtest"
	"github.com/ethereum/go-ethereum/common"
)

type StubBeaconClient struct {
	Headers map[string]*v1.BeaconBlockHeader
	Blobs   map[string][]*deneb.BlobSidecar
}

func (s *StubBeaconClient) BeaconBlockHeader(ctx context.Context, opts *api.BeaconBlockHeaderOpts) (*api.Response[*v1.BeaconBlockHeader], error) {
	header, found := s.Headers[opts.Block]
	if !found {
		return nil, fmt.Errorf("block not found")
	}
	return &api.Response[*v1.BeaconBlockHeader]{
		Data: header,
	}, nil
}

func (s *StubBeaconClient) BlobSidecars(ctx context.Context, opts *api.BlobSidecarsOpts) (*api.Response[[]*deneb.BlobSidecar], error) {
	blobs, found := s.Blobs[opts.Block]
	if !found {
		return nil, fmt.Errorf("block not found")
	}
	return &api.Response[[]*deneb.BlobSidecar]{
		Data: blobs,
	}, nil
}

func NewEmptyStubBeaconClient() *StubBeaconClient {
	return &StubBeaconClient{
		Headers: make(map[string]*v1.BeaconBlockHeader),
		Blobs:   make(map[string][]*deneb.BlobSidecar),
	}
}

func NewDefaultStubBeaconClient(t *testing.T) *StubBeaconClient {
	makeHeader := func(slot uint64, hash, parent common.Hash) *v1.BeaconBlockHeader {
		return &v1.BeaconBlockHeader{
			Root: phase0.Root(hash),
			Header: &phase0.SignedBeaconBlockHeader{
				Message: &phase0.BeaconBlockHeader{
					Slot:       phase0.Slot(slot),
					ParentRoot: phase0.Root(parent),
				},
			},
		}
	}

	headBlobs := blobtest.NewBlobSidecars(t, 6)
	finalizedBlobs := blobtest.NewBlobSidecars(t, 4)

	startSlot := blobtest.StartSlot

	return &StubBeaconClient{
		Headers: map[string]*v1.BeaconBlockHeader{
			// Lookup by hash
			blobtest.OriginBlock.String(): makeHeader(startSlot, blobtest.OriginBlock, common.Hash{9, 9, 9}),
			blobtest.One.String():         makeHeader(startSlot+1, blobtest.One, blobtest.OriginBlock),
			blobtest.Two.String():         makeHeader(startSlot+2, blobtest.Two, blobtest.One),
			blobtest.Three.String():       makeHeader(startSlot+3, blobtest.Three, blobtest.Two),
			blobtest.Four.String():        makeHeader(startSlot+4, blobtest.Four, blobtest.Three),
			blobtest.Five.String():        makeHeader(startSlot+5, blobtest.Five, blobtest.Four),

			// Lookup by identifier
			"head":      makeHeader(startSlot+5, blobtest.Five, blobtest.Four),
			"finalized": makeHeader(startSlot+3, blobtest.Three, blobtest.Two),

			// Lookup by slot
			strconv.FormatUint(startSlot, 10):   makeHeader(startSlot, blobtest.OriginBlock, common.Hash{9, 9, 9}),
			strconv.FormatUint(startSlot+1, 10): makeHeader(startSlot+1, blobtest.One, blobtest.OriginBlock),
			strconv.FormatUint(startSlot+2, 10): makeHeader(startSlot+2, blobtest.Two, blobtest.One),
			strconv.FormatUint(startSlot+3, 10): makeHeader(startSlot+3, blobtest.Three, blobtest.Two),
			strconv.FormatUint(startSlot+4, 10): makeHeader(startSlot+4, blobtest.Four, blobtest.Three),
			strconv.FormatUint(startSlot+5, 10): makeHeader(startSlot+5, blobtest.Five, blobtest.Four),
		},
		Blobs: map[string][]*deneb.BlobSidecar{
			blobtest.OriginBlock.String(): blobtest.NewBlobSidecars(t, 1),
			blobtest.One.String():         blobtest.NewBlobSidecars(t, 2),
			blobtest.Two.String():         blobtest.NewBlobSidecars(t, 0),
			blobtest.Three.String():       finalizedBlobs,
			blobtest.Four.String():        blobtest.NewBlobSidecars(t, 5),
			blobtest.Five.String():        headBlobs,
			"head":                        headBlobs,
			"finalized":                   finalizedBlobs,
		},
	}
}
