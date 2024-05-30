package flags

import (
	"fmt"
	"time"
	"strings"
	
	common "github.com/base-org/blob-archiver/common/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	geth "github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

type ArchiverConfig struct {
	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	BeaconConfig  common.BeaconConfig
	StorageConfig common.StorageConfig
	PollInterval  time.Duration
	OriginBlock   geth.Hash
	ListenAddr    string
}

func (c ArchiverConfig) Check() error {
	if err := c.StorageConfig.Check(); err != nil {
		return err
	}

	if err := c.BeaconConfig.Check(); err != nil {
		return err
	}

	if c.PollInterval == 0 {
		return fmt.Errorf("archiver poll interval must be set")
	}

	if c.OriginBlock == (geth.Hash{}) {
		return fmt.Errorf("invalid origin block %s", c.OriginBlock)
	}

	if c.ListenAddr == "" {
		return fmt.Errorf("archiver listen address must be set")
	}

	return nil
}

func ReadConfig(cliCtx *cli.Context) ArchiverConfig {
	pollInterval, _ := time.ParseDuration(cliCtx.String(ArchiverPollIntervalFlag.Name))

	return ArchiverConfig{
		LogConfig:     oplog.ReadCLIConfig(cliCtx),
		MetricsConfig: opmetrics.ReadCLIConfig(cliCtx),
		BeaconConfig:  common.NewBeaconConfig(cliCtx),
		StorageConfig: common.NewStorageConfig(cliCtx),
		PollInterval:  pollInterval,
		OriginBlock:   geth.HexToHash(strings.Trim(cliCtx.String(ArchiverOriginBlock.Name), "\"")),
		ListenAddr:    cliCtx.String(ArchiverListenAddrFlag.Name),
	}
}
