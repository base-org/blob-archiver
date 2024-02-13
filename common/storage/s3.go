package storage

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/base-org/blob-archiver/common/flags"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage struct {
	s3     *minio.Client
	bucket string
	log    log.Logger
}

func NewS3Storage(cfg flags.S3Config, l log.Logger) (*S3Storage, error) {
	var c *credentials.Credentials
	if cfg.S3CredentialType == flags.S3CredentialStatic {
		c = credentials.NewStaticV4(cfg.AccessKey, cfg.SecretAccessKey, "")
	} else {
		c = credentials.NewIAM("")
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  c,
		Secure: cfg.UseHttps,
	})

	if err != nil {
		return nil, err
	}

	return &S3Storage{
		s3:     client,
		bucket: cfg.Bucket,
		log:    l,
	}, nil
}

func (s *S3Storage) Exists(ctx context.Context, hash common.Hash) (bool, error) {
	_, err := s.s3.StatObject(ctx, s.bucket, hash.String(), minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func (s *S3Storage) Read(ctx context.Context, hash common.Hash) (BlobData, error) {
	res, err := s.s3.GetObject(ctx, s.bucket, hash.String(), minio.GetObjectOptions{})
	if err != nil {
		s.log.Info("unexpected error fetching blob", "hash", hash.String(), "err", err)
		return BlobData{}, ErrStorage
	}
	defer res.Close()
	_, err = res.Stat()
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			s.log.Info("unable to find blob", "hash", hash.String())
			return BlobData{}, ErrNotFound
		} else {
			s.log.Info("unexpected error fetching blob", "hash", hash.String(), "err", err)
			return BlobData{}, ErrStorage
		}
	}

	var data BlobData
	err = json.NewDecoder(res).Decode(&data)
	if err != nil {
		s.log.Warn("error decoding blob", "hash", hash.String(), "err", err)
		return BlobData{}, ErrEncoding
	}

	return data, nil
}

func (s *S3Storage) Write(ctx context.Context, data BlobData) error {
	b, err := json.Marshal(data)
	if err != nil {
		s.log.Warn("error encoding blob", "err", err)
		return ErrEncoding
	}

	reader := bytes.NewReader(b)
	_, err = s.s3.PutObject(ctx, s.bucket, data.Header.BeaconBlockHash.String(), reader, int64(len(b)), minio.PutObjectOptions{
		ContentType: "application/json",
	})

	if err != nil {
		s.log.Warn("error writing blob", "err", err)
		return ErrStorage
	}

	s.log.Info("wrote blob", "hash", data.Header.BeaconBlockHash.String())
	return nil
}
