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
		Name:    "api-listen-address",
		Usage:   "The address to list for new requests on",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "LISTEN_ADDRESS"),
		Value:   "0.0.0.0:8000",
	}
)

func init() {
	Flags = append(Flags, common.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, opmetrics.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, oplog.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, ListenAddressFlag)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
