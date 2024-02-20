package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/base-org/blob-archiver/common/storage"
)

type Format string

const (
	// FormatJson instructs the client to request the response in JSON format
	FormatJson Format = "application/json"
	// FormatSSZ instructs the client to request the response in SSZ format
	FormatSSZ Format = "application/octet-stream"
)

// BlobSidecarClient is a minimal client for fetching sidecars from the blob service. This client is used instead of an
// existing client for two reasons.
// 1) Does not require any endpoints except /eth/v1/blob_sidecar, which is the only endpoint that the Blob API supports
// 2) Exposes implementation details, e.g. status code, as well as allowing us to specify the format
type BlobSidecarClient interface {
	// FetchSidecars fetches the sidecars for a given slot from the blob sidecar API. It returns the HTTP status code and
	// the sidecars.
	FetchSidecars(id string, format Format) (int, storage.BlobSidecars, error)
}

type httpBlobSidecarClient struct {
	url    string
	client *http.Client
}

// NewBlobSidecarClient creates a new BlobSidecarClient that fetches sidecars from the given URL.
func NewBlobSidecarClient(url string) BlobSidecarClient {
	return &httpBlobSidecarClient{
		url:    url,
		client: &http.Client{},
	}
}

func (c *httpBlobSidecarClient) FetchSidecars(id string, format Format) (int, storage.BlobSidecars, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/blob_sidecars/%s", c.url, id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return http.StatusInternalServerError, storage.BlobSidecars{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", string(format))

	response, err := c.client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, storage.BlobSidecars{}, fmt.Errorf("failed to fetch sidecars: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return response.StatusCode, storage.BlobSidecars{}, nil
	}

	defer response.Body.Close()

	var sidecars storage.BlobSidecars
	if format == FormatJson {
		if err := json.NewDecoder(response.Body).Decode(&sidecars); err != nil {
			return response.StatusCode, storage.BlobSidecars{}, fmt.Errorf("failed to decode json response: %w", err)
		}
	} else {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return response.StatusCode, storage.BlobSidecars{}, fmt.Errorf("failed to read response: %w", err)
		}

		s := api.BlobSidecars{}
		if err := s.UnmarshalSSZ(body); err != nil {
			return response.StatusCode, storage.BlobSidecars{}, fmt.Errorf("failed to decode ssz response: %w", err)
		}

		sidecars.Data = s.Sidecars
	}

	return response.StatusCode, sidecars, nil
}
