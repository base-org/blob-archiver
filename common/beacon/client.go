package beacon

import (
	"context"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/base-org/blob-archiver/common/flags"
)

type Client interface {
	client.BeaconBlockHeadersProvider
	client.BlobSidecarsProvider
}

func NewBeaconClient(ctx context.Context, cfg flags.BeaconConfig) (Client, error) {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c, err := http.New(cctx, http.WithAddress(cfg.BeaconUrl), http.WithTimeout(cfg.BeaconClientTimeout))
	if err != nil {
		return nil, err
	}

	return c.(*http.Service), nil
}
