package flags

import (
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
)

const (
	BeaconHttpFlagName              = "l1-beacon-http"
	BeaconHttpClientTimeoutFlagName = "l1-beacon-client-timeout"
	DataStoreFlagName               = "data-store"
	S3EndpointFlagName              = "s3-endpoint"
	S3EndpointHttpsFlagName         = "s3-endpoint-https"
	S3AccessKeyFlagName             = "s3-access-key"
	S3SecretAccessKeyFlagName       = "s3-secret-access-key"
	S3BucketFlagName                = "s3-bucket"
	FileStorageDirectoryFlagName    = "file-directory"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		// Required Flags
		&cli.StringFlag{
			Name:     BeaconHttpFlagName,
			Usage:    "HTTP provider URL for L1 Beacon-node API",
			Required: true,
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "L1_BEACON_HTTP"),
		},
		&cli.StringFlag{
			Name:     DataStoreFlagName,
			Usage:    "The type of data-store, options are [s3, file]",
			Required: true,
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "DATA_STORE"),
		},
		// Optional Flags
		// S3 Data Store Flags
		&cli.StringFlag{
			Name:    S3EndpointFlagName,
			Usage:   "The URL for the S3 bucket (without the scheme http or https specified)",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "S3_ENDPOINT"),
		},
		&cli.BoolFlag{
			Name:    S3EndpointHttpsFlagName,
			Usage:   "Whether to use https for the S3 bucket",
			Value:   true,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "S3_ENDPOINT_HTTPS"),
		},
		&cli.StringFlag{
			Name:    S3AccessKeyFlagName,
			Usage:   "The S3 access key for the bucket",
			Hidden:  true,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "S3_ACCESS_KEY"),
		},
		&cli.StringFlag{
			Name:    S3SecretAccessKeyFlagName,
			Usage:   "The S3 secret access key for the bucket",
			Hidden:  true,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "S3_SECRET_ACCESS_KEY"),
		},
		&cli.StringFlag{
			Name:    S3BucketFlagName,
			Usage:   "The bucket to use",
			Hidden:  true,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "S3_BUCKET"),
		},
		// File Data Store Flags
		&cli.StringFlag{
			Name:    FileStorageDirectoryFlagName,
			Usage:   "The path to the directory to use for storing blobs on the file system",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "FILE_DIRECTORY"),
		},
		// Beacon Client Settings
		&cli.StringFlag{
			Name:    BeaconHttpClientTimeoutFlagName,
			Usage:   "The timeout duration for the beacon client",
			Value:   "10s",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "L1_BEACON_CLIENT_TIMEOUT"),
		},
	}
}
