package service

import (
	"net/http/httptest"
	"testing"

	"github.com/base-org/blob-archiver/archiver/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func setupAPI(t *testing.T) *API {
	logger := testlog.Logger(t, log.LvlInfo)
	m := metrics.NewMetrics()
	return NewAPI(m, logger)
}

func TestHealthHandler(t *testing.T) {
	a := setupAPI(t)

	request := httptest.NewRequest("GET", "/healthz", nil)
	response := httptest.NewRecorder()

	a.router.ServeHTTP(response, request)

	require.Equal(t, 200, response.Code)
}
