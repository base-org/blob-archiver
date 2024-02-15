package storage

import (
	"context"
	"encoding/json"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type FileStorage struct {
	log       log.Logger
	directory string
}

func NewFileStorage(dir string, l log.Logger) *FileStorage {
	return &FileStorage{
		log:       l,
		directory: dir,
	}
}

func (s *FileStorage) Exists(_ context.Context, hash common.Hash) (bool, error) {
	_, err := os.Stat(s.fileName(hash))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *FileStorage) Read(_ context.Context, hash common.Hash) (BlobData, error) {
	data, err := os.ReadFile(s.fileName(hash))
	if err != nil {
		if os.IsNotExist(err) {
			return BlobData{}, ErrNotFound
		}

		return BlobData{}, err
	}
	var result BlobData
	err = json.Unmarshal(data, &result)
	if err != nil {
		s.log.Warn("error decoding blob", "err", err, "hash", hash.String())
		return BlobData{}, ErrMarshaling
	}
	return result, nil
}

func (s *FileStorage) Write(_ context.Context, data BlobData) error {
	b, err := json.Marshal(data)
	if err != nil {
		s.log.Warn("error encoding blob", "err", err)
		return ErrMarshaling
	}
	err = os.WriteFile(s.fileName(data.Header.BeaconBlockHash), b, 0644)
	if err != nil {
		s.log.Warn("error writing blob", "err", err)
		return err
	}

	s.log.Info("wrote blob", "hash", data.Header.BeaconBlockHash.String())
	return nil
}

func (s *FileStorage) fileName(hash common.Hash) string {
	return path.Join(s.directory, hash.String())
}
