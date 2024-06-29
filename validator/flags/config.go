package flags

import (
	"fmt"
	"time"

	common "github.com/base-org/blob-archiver/common/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

type ValidatorConfig struct {
	LogConfig       oplog.CLIConfig
	BeaconConfig    common.BeaconConfig
	BlobConfig      common.BeaconConfig
	ValidateFormats []string
	NumBlocks       int
}

func (c ValidatorConfig) Check() error {
	if err := c.BeaconConfig.Check(); err != nil {
		return fmt.Errorf("beacon config check failed: %w", err)
	}

	if err := c.BlobConfig.Check(); err != nil {
		return fmt.Errorf("blob config check failed: %w", err)
	}

	if c.NumBlocks <= 0 {
		return fmt.Errorf("number of blocks must be greater than 0")
	}

	if len(c.ValidateFormats) == 0 || len(c.ValidateFormats) > 2 {
		return fmt.Errorf("no formats to validate, please specify formats [json,ssz]")
	}

	seen := make(map[string]struct{})
	for _, format := range c.ValidateFormats {
		if format != "json" && format != "ssz" {
			return fmt.Errorf("invalid format %v, please specify formats [json,ssz]", format)
		}
		if _, ok := seen[format]; ok {
			return fmt.Errorf("duplicate format %v", format)
		}
		seen[format] = struct{}{}
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
		NumBlocks:       cliCtx.Int(NumBlocksClientFlag.Name),
		ValidateFormats: cliCtx.StringSlice(ValidateFormatsFlag.Name),
	}
}
