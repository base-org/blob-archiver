package version

import "fmt"

var (
	GitCommit  = ""
	APIVersion Version
)

func init() {
	commit := GitCommit
	if commit == "" {
		commit = "unknown"
	}

	APIVersion = Version{
		Data: struct {
			Version string `json:"version"`
		}{
			Version: fmt.Sprintf("Blob Archiver API/%s", commit),
		},
	}
}

type Version struct {
	Data struct {
		Version string `json:"version"`
	} `json:"data"`
}
