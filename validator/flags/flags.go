package flags

import (
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

const EnvVarPrefix = "BLOB_VALIDATOR"

var (
	BeaconClientTimeoutFlag = &cli.StringFlag{
		Name:    "beacon-client-timeout",
		Usage:   "The timeout duration for the beacon client",
		Value:   "10s",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "CLIENT_TIMEOUT"),
	}
	L1BeaconClientUrlFlag = &cli.StringFlag{
		Name:     "l1-beacon-http",
		Usage:    "URL for a L1 Beacon-node API",
		Required: true,
		EnvVars:  opservice.PrefixEnvVar(EnvVarPrefix, "L1_BEACON_HTTP"),
	}
	BlobApiClientUrlFlag = &cli.StringFlag{
		Name:     "blob-api-http",
		Usage:    "URL for a Blob API",
		Required: true,
		EnvVars:  opservice.PrefixEnvVar(EnvVarPrefix, "BLOB_API_HTTP"),
	}
	NumBlocksClientFlag = &cli.IntFlag{
		Name:     "num-blocks",
		Usage:    "The number of blocks to read blob data for",
		Value:    600,
		Required: true,
		EnvVars:  opservice.PrefixEnvVar(EnvVarPrefix, "NUM_BLOCKS"),
	}
)

func init() {
	Flags = append(Flags, oplog.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, BeaconClientTimeoutFlag, L1BeaconClientUrlFlag, BlobApiClientUrlFlag, NumBlocksClientFlag)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
