package version

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	GitCommit  = ""
	APIVersion eth.APIVersionResponse
)

func init() {
	commit := GitCommit
	if commit == "" {
		commit = "unknown"
	}

	APIVersion = eth.APIVersionResponse{
		Data: eth.VersionInformation{
			Version: fmt.Sprintf("Blob Archiver API/%s", commit),
		},
	}
}
