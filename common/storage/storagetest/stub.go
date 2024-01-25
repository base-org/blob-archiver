package storagetest

import (
	"context"
	"testing"

	"github.com/base-org/blob-archiver/common/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type TestFileStorage struct {
	*storage.FileStorage
}

func NewTestFileStorage(t *testing.T, l log.Logger) *TestFileStorage {
	dir := t.TempDir()
	return &TestFileStorage{
		FileStorage: storage.NewFileStorage(dir, l),
	}
}

func (fs *TestFileStorage) CheckExistsOrFail(t *testing.T, hash common.Hash) {
	exists, err := fs.Exists(context.Background(), hash)
	require.NoError(t, err)
	require.True(t, exists)
}

func (fs *TestFileStorage) CheckNotExistsOrFail(t *testing.T, hash common.Hash) {
	exists, err := fs.Exists(context.Background(), hash)
	require.NoError(t, err)
	require.False(t, exists)
}

func (fs *TestFileStorage) WriteOrFail(t *testing.T, data storage.BlobData) {
	err := fs.Write(context.Background(), data)
	require.NoError(t, err)
}

func (fs *TestFileStorage) ReadOrFail(t *testing.T, hash common.Hash) storage.BlobData {
	data, err := fs.Read(context.Background(), hash)
	require.NoError(t, err)
	require.NotNil(t, data)
	return data
}
