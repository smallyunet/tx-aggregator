package utils_test

import (
	"testing"
	"tx-aggregator/config"
	"tx-aggregator/utils"

	"github.com/stretchr/testify/assert"
)

func setupTestConfig() {
	config.AppConfig.ChainNames = map[string]int64{
		"ETH": 1,
		"BSC": 56,
		"op":  10,
	}

	config.AppConfig.Ankr.ChainIDs = map[string]int64{
		"ETH":  1,
		"AVAX": 43114,
		"arb":  42161,
	}
}

func TestChainIDByName(t *testing.T) {
	setupTestConfig()

	tests := []struct {
		name      string
		input     string
		expected  int64
		expectErr bool
	}{
		{"valid name uppercase", "ETH", 1, false},
		{"valid name lowercase", "bsc", 56, false},
		{"valid name mixed case", "Op", 10, false},
		{"invalid name", "unknown", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := utils.ChainIDByName(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, id)
			}
		})
	}
}

func TestChainNameByID(t *testing.T) {
	setupTestConfig()

	tests := []struct {
		name      string
		input     int64
		expected  string
		expectErr bool
	}{
		{"valid ID ETH", 1, "ETH", false},
		{"valid ID BSC", 56, "BSC", false},
		{"invalid ID", 999, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainName, err := utils.ChainNameByID(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, chainName)
			}
		})
	}
}

func TestAnkrChainIDByName(t *testing.T) {
	setupTestConfig()

	tests := []struct {
		name      string
		input     string
		expected  int64
		expectErr bool
	}{
		{"valid ETH", "eth", 1, false},
		{"valid AVAX", "AVAX", 43114, false},
		{"valid ARB mixed case", "ArB", 42161, false},
		{"invalid chain", "polygon", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := utils.AnkrChainIDByName(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, id)
			}
		})
	}
}

func TestAnkrChainNameByID(t *testing.T) {
	setupTestConfig()

	tests := []struct {
		name      string
		input     int64
		expected  string
		expectErr bool
	}{
		{"valid ETH", 1, "ETH", false},
		{"valid AVAX", 43114, "AVAX", false},
		{"invalid chain ID", 100000, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := utils.AnkrChainNameByID(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, name)
			}
		})
	}
}
