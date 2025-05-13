package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"tx-aggregator/config"
	"tx-aggregator/types"
)

// setupTestConfig injects test config with mock ChainNames
func setupTestConfig() {
	cfg := config.Current()
	cfg.ChainNames = map[string]int64{
		"ETH": 1,
		"BSC": 56,
	}
	config.SetCurrentConfig(cfg)
}

func TestParseTransactionQueryParams(t *testing.T) {
	setupTestConfig()

	tests := []struct {
		name           string
		query          string
		expectedError  string
		expectedResult *types.TransactionQueryParams
	}{
		{
			name:          "missing address",
			query:         "",
			expectedError: "address parameter is required",
		},
		{
			name:          "invalid address format",
			query:         "?address=invalid",
			expectedError: "invalid address: invalid",
		},
		{
			name:  "no chainName, use all chains",
			query: "?address=0x0123456789ABCDEF0123456789ABCDEF01234567",
			expectedResult: &types.TransactionQueryParams{
				Address:      "0x0123456789abcdef0123456789abcdef01234567",
				TokenAddress: "",
				ChainNames:   []string{"BSC", "ETH"}, // sorted
			},
		},
		{
			name:  "valid address and valid chain names mixed case",
			query: "?address=0x0123456789ABCDEF0123456789ABCDEF01234567&chainName=eth,bsc",
			expectedResult: &types.TransactionQueryParams{
				Address:      "0x0123456789abcdef0123456789abcdef01234567",
				TokenAddress: "",
				ChainNames:   []string{"BSC", "ETH"}, // sorted
			},
		},
		{
			name:          "contains unknown chain name",
			query:         "?address=0x0123456789abcdef0123456789abcdef01234567&chainName=eth,xxx",
			expectedError: "unknown chain names: XXX",
		},
		{
			name:          "invalid token address",
			query:         "?address=0x0123456789abcdef0123456789abcdef01234567&tokenAddress=abc",
			expectedError: "invalid token address: abc",
		},
		{
			name:  "valid token address native",
			query: "?address=0x0123456789abcdef0123456789abcdef01234567&tokenAddress=native",
			expectedResult: &types.TransactionQueryParams{
				Address:      "0x0123456789abcdef0123456789abcdef01234567",
				TokenAddress: "native",
				ChainNames:   []string{"BSC", "ETH"}, // sorted
			},
		},
		{
			name:  "valid token address ethereum",
			query: "?address=0x0123456789abcdef0123456789abcdef01234567&tokenAddress=0x000000000000000000000000000000000000DEAD",
			expectedResult: &types.TransactionQueryParams{
				Address:      "0x0123456789abcdef0123456789abcdef01234567",
				TokenAddress: "0x000000000000000000000000000000000000dead",
				ChainNames:   []string{"BSC", "ETH"}, // sorted
			},
		},
		{
			name:  "chainName with spaces and mixed case",
			query: "?address=0x0123456789abcdef0123456789abcdef01234567&chainName=%20ETH%20,%20bsc%20",
			expectedResult: &types.TransactionQueryParams{
				Address:      "0x0123456789abcdef0123456789abcdef01234567",
				TokenAddress: "",
				ChainNames:   []string{"BSC", "ETH"}, // sorted
			},
		},
		{
			name:  "tokenAddress upper case, ensure lower",
			query: "?address=0x0123456789abcdef0123456789abcdef01234567&tokenAddress=0X000000000000000000000000000000000000DEAD",
			expectedResult: &types.TransactionQueryParams{
				Address:      "0x0123456789abcdef0123456789abcdef01234567",
				TokenAddress: "0x000000000000000000000000000000000000dead",
				ChainNames:   []string{"BSC", "ETH"}, // sorted
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var result *types.TransactionQueryParams
			var handlerErr error

			app.Get("/tx", func(c *fiber.Ctx) error {
				result, handlerErr = parseTransactionQueryParams(c)
				return nil
			})

			req := httptest.NewRequest(http.MethodGet, "/tx"+tt.query, nil)
			_, _ = app.Test(req)

			if tt.expectedError != "" {
				assert.Nil(t, result)
				assert.Error(t, handlerErr)
				assert.EqualError(t, handlerErr, tt.expectedError)
			} else {
				assert.NoError(t, handlerErr)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
