package service

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/base-org/blob-archiver/archiver/flags"
	"github.com/base-org/blob-archiver/archiver/metrics"
	"github.com/base-org/blob-archiver/common/storage/storagetest"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func setupAPI(t *testing.T) (*API, *storagetest.TestFileStorage) {
	logger := testlog.Logger(t, log.LvlInfo)
	m := metrics.NewMetrics()
	fs := storagetest.NewTestFileStorage(t, logger)
	archiver, err := NewArchiver(logger, flags.ArchiverConfig{
		PollInterval: 10 * time.Second,
	}, fs, nil, m)
	require.NoError(t, err)
	return NewAPI(m, logger, archiver), fs
}

func TestHealthHandler(t *testing.T) {
	a, _ := setupAPI(t)

	request := httptest.NewRequest("GET", "/healthz", nil)
	response := httptest.NewRecorder()

	a.router.ServeHTTP(response, request)

	require.Equal(t, 200, response.Code)
}

func TestRearchiveHandler(t *testing.T) {
	a, _ := setupAPI(t)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		error          string
	}{
		{
			name:           "should fail with no params",
			path:           "/rearchive",
			expectedStatus: 400,
			error:          "invalid from param: must provide param",
		},
		{
			name:           "should fail with missing to param",
			path:           "/rearchive?from=1",
			expectedStatus: 400,
			error:          "invalid to param: must provide param",
		},
		{
			name:           "should fail with missing from param",
			path:           "/rearchive?to=1",
			expectedStatus: 400,
			error:          "invalid from param: must provide param",
		},
		{
			name:           "should fail with invalid from param",
			path:           "/rearchive?from=blah&to=1",
			expectedStatus: 400,
			error:          "invalid from param: invalid slot: \"blah\"",
		},
		{
			name:           "should fail with invalid to param",
			path:           "/rearchive?from=1&to=blah",
			expectedStatus: 400,
			error:          "invalid to param: invalid slot: \"blah\"",
		},
		{
			name:           "should fail with to greater than equal to from",
			path:           "/rearchive?from=2&to=1",
			expectedStatus: 400,
			error:          "invalid range: from 2 to 1",
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", test.path, nil)
			response := httptest.NewRecorder()

			a.router.ServeHTTP(response, request)

			require.Equal(t, test.expectedStatus, response.Code)

			var errResponse rearchiveResponse
			err := json.NewDecoder(response.Body).Decode(&errResponse)
			require.NoError(t, err)

			if test.error != "" {
				require.Equal(t, errResponse.Error, test.error)
			}
		})
	}
}
