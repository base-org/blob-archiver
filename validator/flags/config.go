package flags

import (
	"fmt"
	"time"

	common "github.com/base-org/blob-archiver/common/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

type ValidatorConfig struct {
	LogConfig    oplog.CLIConfig
	BeaconConfig common.BeaconConfig
	BlobConfig   common.BeaconConfig
}

func (c ValidatorConfig) Check() error {
	if err := c.BeaconConfig.Check(); err != nil {
		return fmt.Errorf("beacon config check failed: %w", err)
	}

	if err := c.BlobConfig.Check(); err != nil {
		return fmt.Errorf("blob config check failed: %w", err)
	}

	return nil
}

func ReadConfig(cliCtx *cli.Context) ValidatorConfig {
	timeout, _ := time.ParseDuration(cliCtx.String(BeaconClientTimeoutFlag.Name))

	return ValidatorConfig{
		LogConfig: oplog.ReadCLIConfig(cliCtx),
		BeaconConfig: common.BeaconConfig{
			BeaconURL:           cliCtx.String(L1BeaconClientUrlFlag.Name),
			BeaconClientTimeout: timeout,
		},
		BlobConfig: common.BeaconConfig{
			BeaconURL:           cliCtx.String(BlobApiClientUrlFlag.Name),
			BeaconClientTimeout: timeout,
		},
	}
}
