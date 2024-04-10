package main

import (
	"context"
	"fmt"
	"os"

	"github.com/base-org/blob-archiver/common/beacon"
	"github.com/base-org/blob-archiver/validator/flags"
	"github.com/base-org/blob-archiver/validator/service"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var (
	Version   = "v0.0.1"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "blob-validator"
	app.Usage = "Job that checks the validity of blobs"
	app.Description = "The blob-validator is a job that checks the validity of blobs"
	app.Action = cliapp.LifecycleCmd(Main())

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

// Main is the entrypoint into the API.
// This method returns a cliapp.LifecycleAction, to create an op-service CLI-lifecycle-managed API Server.
func Main() cliapp.LifecycleAction {
	return func(cliCtx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		cfg := flags.ReadConfig(cliCtx)
		if err := cfg.Check(); err != nil {
			return nil, fmt.Errorf("config check failed: %w", err)
		}

		l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogConfig)
		oplog.SetGlobalLogHandler(l.Handler())
		opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, l)

		headerClient, err := beacon.NewBeaconClient(cliCtx.Context, cfg.BeaconConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create beacon client: %w", err)
		}

		beaconClient := service.NewBlobSidecarClient(cfg.BeaconConfig.BeaconURL)
		blobClient := service.NewBlobSidecarClient(cfg.BlobConfig.BeaconURL)

		return service.NewValidator(l, headerClient, beaconClient, blobClient, closeApp), nil
	}
}
