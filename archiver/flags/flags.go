package flags

import (
	common "github.com/base-org/blob-archiver/common/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/urfave/cli/v2"
)

const EnvVarPrefix = "BLOB_ARCHIVER"

var (
	ArchiverPollIntervalFlag = &cli.StringFlag{
		Name:    "archiver-poll-interval",
		Usage:   "The interval at which the archiver polls for new blobs",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "ARCHIVER_POLL_INTERVAL"),
		Value:   "6s",
	}
	ArchiverOriginBlock = &cli.StringFlag{
		Name:     "archiver-origin-block",
		Usage:    "The latest block hash that the archiver will walk back to",
		Required: true,
		EnvVars:  opservice.PrefixEnvVar(EnvVarPrefix, "ORIGIN_BLOCK"),
	}
	ArchiverListenAddrFlag = &cli.StringFlag{
		Name:    "archiver-listen-address",
		Usage:   "The address to list for new requests on",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "LISTEN_ADDRESS"),
		Value:   "0.0.0.0:8000",
	}
)

func init() {
	Flags = append(Flags, common.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, opmetrics.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, oplog.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, ArchiverPollIntervalFlag, ArchiverOriginBlock, ArchiverListenAddrFlag)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
