package flags

import (
	common "github.com/base-org/blob-archiver/common/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/urfave/cli/v2"
)

const EnvVarPrefix = "BLOB_API"

var (
	ListenAddressFlag = &cli.StringFlag{
		Name:    "api-list-address",
		Usage:   "The address to list for new requests on",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "LISTEN_ADDRESS"),
		Value:   "0.0.0.0:8000",
	}
)

func init() {
	var flags []cli.Flag

	flags = append(flags, common.CLIFlags(EnvVarPrefix)...)
	flags = append(flags, opmetrics.CLIFlags(EnvVarPrefix)...)
	flags = append(flags, oplog.CLIFlags(EnvVarPrefix)...)
	flags = append(flags, ListenAddressFlag)

	Flags = flags
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
