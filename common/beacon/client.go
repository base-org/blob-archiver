package beacon

import (
	"context"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/base-org/blob-archiver/common/flags"
)

// Client is an interface that wraps the go-eth-2 interfaces that the blob archiver and api require.
type Client interface {
	client.BeaconBlockHeadersProvider
	client.BlobSidecarsProvider
}

// NewBeaconClient returns a new HTTP beacon client.
func NewBeaconClient(ctx context.Context, cfg flags.BeaconConfig) (Client, error) {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c, err := http.New(cctx, http.WithAddress(cfg.BeaconURL), http.WithTimeout(cfg.BeaconClientTimeout))
	if err != nil {
		return nil, err
	}

	return c.(*http.Service), nil
}
