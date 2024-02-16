package service

import (
	"context"
	"testing"
	"time"

	"github.com/base-org/blob-archiver/archiver/flags"
	"github.com/base-org/blob-archiver/archiver/metrics"
	"github.com/base-org/blob-archiver/common/beacon/beacontest"
	"github.com/base-org/blob-archiver/common/blobtest"
	"github.com/base-org/blob-archiver/common/storage"
	"github.com/base-org/blob-archiver/common/storage/storagetest"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T, beacon *beacontest.StubBeaconClient) (*ArchiverService, *storagetest.TestFileStorage) {
	l := testlog.Logger(t, log.LvlInfo)
	fs := storagetest.NewTestFileStorage(t, l)
	m := metrics.NewMetrics()

	svc, err := NewService(l, flags.ArchiverConfig{
		PollInterval: 5 * time.Second,
		OriginBlock:  blobtest.OriginBlock,
	}, NewAPI(m, l), fs, beacon, m)
	require.NoError(t, err)
	return svc, fs
}

func TestArchiver_FetchAndPersist(t *testing.T) {
	svc, fs := setup(t, beacontest.NewDefaultStubBeaconClient(t))

	fs.CheckNotExistsOrFail(t, blobtest.OriginBlock)

	header, alreadyExists, err := svc.persistBlobsForBlockToS3(context.Background(), blobtest.OriginBlock.String())
	require.False(t, alreadyExists)
	require.NoError(t, err)
	require.NotNil(t, header)
	require.Equal(t, blobtest.OriginBlock.String(), common.Hash(header.Root).String())

	fs.CheckExistsOrFail(t, blobtest.OriginBlock)

	header, alreadyExists, err = svc.persistBlobsForBlockToS3(context.Background(), blobtest.OriginBlock.String())
	require.True(t, alreadyExists)
	require.NoError(t, err)
	require.NotNil(t, header)
	require.Equal(t, blobtest.OriginBlock.String(), common.Hash(header.Root).String())

	fs.CheckExistsOrFail(t, blobtest.OriginBlock)
}

func TestArchiver_BackfillToOrigin(t *testing.T) {
	beacon := beacontest.NewDefaultStubBeaconClient(t)
	svc, fs := setup(t, beacon)

	// We have the current head, which is block 5 written to storage
	err := fs.Write(context.Background(), storage.BlobData{
		Header: storage.Header{
			BeaconBlockHash: blobtest.Five,
		},
		BlobSidecars: storage.BlobSidecars{
			Data: beacon.Blobs[blobtest.Five.String()],
		},
	})
	require.NoError(t, err)
	// We expect to backfill all blocks to the origin
	expectedBlobs := []common.Hash{blobtest.Four, blobtest.Three, blobtest.Two, blobtest.One, blobtest.OriginBlock}

	for _, blob := range expectedBlobs {
		fs.CheckNotExistsOrFail(t, blob)
	}

	svc.backfillBlobs(context.Background(), beacon.Headers[blobtest.Five.String()])

	for _, blob := range expectedBlobs {
		fs.CheckExistsOrFail(t, blob)
		data := fs.ReadOrFail(t, blob)
		require.Equal(t, data.BlobSidecars.Data, beacon.Blobs[blob.String()])
	}
}

func TestArchiver_BackfillToExistingBlock(t *testing.T) {
	beacon := beacontest.NewDefaultStubBeaconClient(t)
	svc, fs := setup(t, beacon)

	// We have the current head, which is block 5 written to storage
	err := fs.Write(context.Background(), storage.BlobData{
		Header: storage.Header{
			BeaconBlockHash: blobtest.Five,
		},
		BlobSidecars: storage.BlobSidecars{
			Data: beacon.Blobs[blobtest.Five.String()],
		},
	})
	require.NoError(t, err)

	// We also have block 1 written to storage
	err = fs.Write(context.Background(), storage.BlobData{
		Header: storage.Header{
			BeaconBlockHash: blobtest.One,
		},
		BlobSidecars: storage.BlobSidecars{
			Data: beacon.Blobs[blobtest.One.String()],
		},
	})
	require.NoError(t, err)

	// We expect to backfill all blobs between 5 and 1
	expectedBlobs := []common.Hash{blobtest.Four, blobtest.Three, blobtest.Two}

	for _, blob := range expectedBlobs {
		exists, err := fs.Exists(context.Background(), blob)
		require.NoError(t, err)
		require.False(t, exists)
	}

	svc.backfillBlobs(context.Background(), beacon.Headers[blobtest.Five.String()])

	for _, blob := range expectedBlobs {
		exists, err := fs.Exists(context.Background(), blob)
		require.NoError(t, err)
		require.True(t, exists)

		data, err := fs.Read(context.Background(), blob)
		require.NoError(t, err)
		require.NotNil(t, data)
		require.Equal(t, data.BlobSidecars.Data, beacon.Blobs[blob.String()])
	}
}

func TestArchiver_LatestStopsAtExistingBlock(t *testing.T) {
	beacon := beacontest.NewDefaultStubBeaconClient(t)
	svc, fs := setup(t, beacon)

	// 5 is the current head, if three already exists, we should write 5 and 4 and stop at three
	fs.WriteOrFail(t, storage.BlobData{
		Header: storage.Header{
			BeaconBlockHash: blobtest.Three,
		},
		BlobSidecars: storage.BlobSidecars{
			Data: beacon.Blobs[blobtest.Three.String()],
		},
	})

	fs.CheckNotExistsOrFail(t, blobtest.Five)
	fs.CheckNotExistsOrFail(t, blobtest.Four)
	fs.CheckExistsOrFail(t, blobtest.Three)

	svc.processBlocksUntilKnownBlock(context.Background())

	fs.CheckExistsOrFail(t, blobtest.Five)
	five := fs.ReadOrFail(t, blobtest.Five)
	require.Equal(t, five.Header.BeaconBlockHash, blobtest.Five)
	require.Equal(t, five.BlobSidecars.Data, beacon.Blobs[blobtest.Five.String()])

	fs.CheckExistsOrFail(t, blobtest.Four)
	four := fs.ReadOrFail(t, blobtest.Four)
	require.Equal(t, four.Header.BeaconBlockHash, blobtest.Four)
	require.Equal(t, five.BlobSidecars.Data, beacon.Blobs[blobtest.Five.String()])

	fs.CheckExistsOrFail(t, blobtest.Three)
	three := fs.ReadOrFail(t, blobtest.Three)
	require.Equal(t, three.Header.BeaconBlockHash, blobtest.Three)
	require.Equal(t, five.BlobSidecars.Data, beacon.Blobs[blobtest.Five.String()])
}

func TestArchiver_LatestNoNewData(t *testing.T) {
	beacon := beacontest.NewDefaultStubBeaconClient(t)
	svc, fs := setup(t, beacon)

	// 5 is the current head, if 5 already exists, this should be a no-op
	fs.WriteOrFail(t, storage.BlobData{
		Header: storage.Header{
			BeaconBlockHash: common.Hash(beacon.Headers["head"].Root),
		},
		BlobSidecars: storage.BlobSidecars{
			Data: beacon.Blobs[blobtest.Three.String()],
		},
	})

	fs.CheckExistsOrFail(t, blobtest.Five)
	fs.CheckNotExistsOrFail(t, blobtest.Four)

	svc.processBlocksUntilKnownBlock(context.Background())

	fs.CheckExistsOrFail(t, blobtest.Five)
	fs.CheckNotExistsOrFail(t, blobtest.Four)
}

func TestArchiver_LatestConsumesNewBlocks(t *testing.T) {
	beacon := beacontest.NewDefaultStubBeaconClient(t)
	svc, fs := setup(t, beacon)

	// set current head to 4, and write four
	beacon.Headers["head"] = beacon.Headers[blobtest.Four.String()]
	fs.WriteOrFail(t, storage.BlobData{
		Header: storage.Header{
			BeaconBlockHash: common.Hash(beacon.Headers[blobtest.Four.String()].Root),
		},
		BlobSidecars: storage.BlobSidecars{
			Data: beacon.Blobs[blobtest.Four.String()],
		},
	})

	svc.processBlocksUntilKnownBlock(context.Background())

	// No new data (5) is written and latest stops at known block (4), so 3 should not exist
	fs.CheckNotExistsOrFail(t, blobtest.Five)
	fs.CheckExistsOrFail(t, blobtest.Four)
	fs.CheckNotExistsOrFail(t, blobtest.Three)

	// set current head to 5, and check it fetches new data
	beacon.Headers["head"] = beacon.Headers[blobtest.Five.String()]

	svc.processBlocksUntilKnownBlock(context.Background())
	fs.CheckExistsOrFail(t, blobtest.Five)
	fs.CheckExistsOrFail(t, blobtest.Four)
	fs.CheckNotExistsOrFail(t, blobtest.Three)
}

func TestArchiver_LatestStopsAtOrigin(t *testing.T) {
	beacon := beacontest.NewDefaultStubBeaconClient(t)
	svc, fs := setup(t, beacon)

	// 5 is the current head, if origin already exists, we should stop at origin
	fs.WriteOrFail(t, storage.BlobData{
		Header: storage.Header{
			BeaconBlockHash: blobtest.OriginBlock,
		},
		BlobSidecars: storage.BlobSidecars{
			Data: beacon.Blobs[blobtest.OriginBlock.String()],
		},
	})

	// Should write all blocks back to Origin
	toWrite := []common.Hash{blobtest.Five, blobtest.Four, blobtest.Three, blobtest.Two, blobtest.One}
	for _, hash := range toWrite {
		fs.CheckNotExistsOrFail(t, hash)
	}

	svc.processBlocksUntilKnownBlock(context.Background())

	for _, hash := range toWrite {
		fs.CheckExistsOrFail(t, hash)
		data := fs.ReadOrFail(t, hash)
		require.Equal(t, data.BlobSidecars.Data, beacon.Blobs[hash.String()])
	}
}

func TestArchiver_LatestRetriesOnFailure(t *testing.T) {
	beacon := beacontest.NewDefaultStubBeaconClient(t)
	svc, fs := setup(t, beacon)

	// 5 is the current head, if three already exists, we should write 5 and 4 and stop at three
	fs.WriteOrFail(t, storage.BlobData{
		Header: storage.Header{
			BeaconBlockHash: blobtest.Three,
		},
		BlobSidecars: storage.BlobSidecars{
			Data: beacon.Blobs[blobtest.Three.String()],
		},
	})

	fs.CheckNotExistsOrFail(t, blobtest.Five)
	fs.CheckNotExistsOrFail(t, blobtest.Four)
	fs.CheckExistsOrFail(t, blobtest.Three)

	// One failure is retried
	fs.WritesFailTimes(1)
	svc.processBlocksUntilKnownBlock(context.Background())

	fs.CheckExistsOrFail(t, blobtest.Five)
	fs.CheckExistsOrFail(t, blobtest.Four)
	fs.CheckExistsOrFail(t, blobtest.Three)
}

func TestArchiver_LatestHaltsOnPersistentError(t *testing.T) {
	beacon := beacontest.NewDefaultStubBeaconClient(t)
	svc, fs := setup(t, beacon)

	// 5 is the current head, if three already exists, we should write 5 and 4 and stop at three
	fs.WriteOrFail(t, storage.BlobData{
		Header: storage.Header{
			BeaconBlockHash: blobtest.Three,
		},
		BlobSidecars: storage.BlobSidecars{
			Data: beacon.Blobs[blobtest.Three.String()],
		},
	})

	fs.CheckNotExistsOrFail(t, blobtest.Five)
	fs.CheckNotExistsOrFail(t, blobtest.Four)
	fs.CheckExistsOrFail(t, blobtest.Three)

	// One failure is retried
	fs.WritesFailTimes(maxLiveAttempts + 1)
	svc.processBlocksUntilKnownBlock(context.Background())

	fs.CheckNotExistsOrFail(t, blobtest.Five)
	fs.CheckNotExistsOrFail(t, blobtest.Four)
	fs.CheckExistsOrFail(t, blobtest.Three)
}
