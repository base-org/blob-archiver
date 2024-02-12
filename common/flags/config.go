package flags

import (
	"errors"
	"time"

	"github.com/urfave/cli/v2"
)

type DataStorage string
type S3CredentialType string

const (
	DataStorageUnknown  DataStorage      = "unknown"
	DataStorageS3       DataStorage      = "s3"
	DataStorageFile     DataStorage      = "file"
	S3CredentialUnknown S3CredentialType = "unknown"
	S3CredentialStatic  S3CredentialType = "static"
	S3CredentialIAM     S3CredentialType = "iam"
)

type S3Config struct {
	Endpoint string
	UseHttps bool
	Bucket   string

	S3CredentialType S3CredentialType
	AccessKey        string
	SecretAccessKey  string
}

func (c S3Config) check() error {
	if c.Endpoint == "" {
		return errors.New("s3 endpoint must be set")
	}

	if c.S3CredentialType == S3CredentialUnknown {
		return errors.New("s3 credential type must be set")
	}

	if c.S3CredentialType == S3CredentialStatic {
		if c.AccessKey == "" {
			return errors.New("s3 access key must be set")
		}

		if c.SecretAccessKey == "" {
			return errors.New("s3 secret access key must be set")
		}
	}

	if c.Bucket == "" {
		return errors.New("s3 bucket must be set")
	}

	return nil
}

type BeaconConfig struct {
	BeaconUrl           string
	BeaconClientTimeout time.Duration
}

type StorageConfig struct {
	DataStorageType      DataStorage
	S3Config             S3Config
	FileStorageDirectory string
}

func NewBeaconConfig(cliCtx *cli.Context) BeaconConfig {
	timeout, _ := time.ParseDuration(cliCtx.String(BeaconHttpClientTimeoutFlagName))

	return BeaconConfig{
		BeaconUrl:           cliCtx.String(BeaconHttpFlagName),
		BeaconClientTimeout: timeout,
	}
}

func NewStorageConfig(cliCtx *cli.Context) StorageConfig {
	return StorageConfig{
		DataStorageType:      toDataStorage(cliCtx.String(DataStoreFlagName)),
		S3Config:             readS3Config(cliCtx),
		FileStorageDirectory: cliCtx.String(FileStorageDirectoryFlagName),
	}
}

func toDataStorage(s string) DataStorage {
	if s == string(DataStorageS3) {
		return DataStorageS3
	}

	if s == string(DataStorageFile) {
		return DataStorageFile
	}

	return DataStorageUnknown
}

func readS3Config(ctx *cli.Context) S3Config {
	return S3Config{
		Endpoint:         ctx.String(S3EndpointFlagName),
		AccessKey:        ctx.String(S3AccessKeyFlagName),
		SecretAccessKey:  ctx.String(S3SecretAccessKeyFlagName),
		UseHttps:         ctx.Bool(S3EndpointHttpsFlagName),
		Bucket:           ctx.String(S3BucketFlagName),
		S3CredentialType: toS3CredentialType(ctx.String(S3CredentialTypeFlagName)),
	}
}

func toS3CredentialType(s string) S3CredentialType {
	if s == string(S3CredentialStatic) {
		return S3CredentialStatic
	} else if s == string(S3CredentialIAM) {
		return S3CredentialIAM
	}
	return S3CredentialUnknown
}

func (c BeaconConfig) Check() error {
	if c.BeaconUrl == "" {
		return errors.New("beacon url must be set")
	}

	if c.BeaconClientTimeout == 0 {
		return errors.New("beacon client timeout must be set")
	}

	return nil
}

func (c StorageConfig) Check() error {
	if c.DataStorageType == DataStorageUnknown {
		return errors.New("unknown data-storage type")
	}

	if c.DataStorageType == DataStorageS3 {
		if err := c.S3Config.check(); err != nil {
			return err
		}
	} else if c.DataStorageType == DataStorageFile && c.FileStorageDirectory == "" {
		return errors.New("file storage directory must be set")
	}

	return nil
}
