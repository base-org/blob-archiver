package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"github.com/base-org/blob-archiver/common/flags"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage struct {
	s3       *minio.Client
	bucket   string
	log      log.Logger
	compress bool
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
		s3:       client,
		bucket:   cfg.Bucket,
		log:      l,
		compress: cfg.Compress,
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

	// TODO: We may need to decode if it's gzipped

	var data BlobData
	err = json.NewDecoder(res).Decode(&data)
	if err != nil {
		s.log.Warn("error decoding blob", "hash", hash.String(), "err", err)
		return BlobData{}, ErrMarshaling
	}

	return data, nil
}

func (s *S3Storage) Write(ctx context.Context, data BlobData) error {
	b, err := json.Marshal(data)
	if err != nil {
		s.log.Warn("error encoding blob", "err", err)
		return ErrMarshaling
	}

	options := minio.PutObjectOptions{
		ContentType: "application/json",
	}

	if s.compress {
		b, err = compress(b)
		if err != nil {
			s.log.Warn("error compressing blob", "err", err)
			return ErrCompress
		}
		options.ContentEncoding = "gzip"
	}

	reader := bytes.NewReader(b)

	_, err = s.s3.PutObject(ctx, s.bucket, data.Header.BeaconBlockHash.String(), reader, int64(len(b)), options)

	if err != nil {
		s.log.Warn("error writing blob", "err", err)
		return ErrStorage
	}

	s.log.Info("wrote blob", "hash", data.Header.BeaconBlockHash.String())
	return nil
}

func compress(in []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(in)
	if err != nil {
		return nil, err
	}
	err = gz.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
