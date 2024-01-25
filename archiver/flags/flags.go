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
		Usage:    "The lastest block hash that the archiver will walk back to",
		Required: true,
		EnvVars:  opservice.PrefixEnvVar(EnvVarPrefix, "ORIGIN_BLOCK"),
	}
)

func init() {
	var flags []cli.Flag

	flags = append(flags, common.CLIFlags(EnvVarPrefix)...)
	flags = append(flags, opmetrics.CLIFlags(EnvVarPrefix)...)
	flags = append(flags, oplog.CLIFlags(EnvVarPrefix)...)
	flags = append(flags, ArchiverPollIntervalFlag, ArchiverOriginBlock)

	Flags = flags
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
