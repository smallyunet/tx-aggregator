package utils_test

import (
	"testing"
	"time"
	"tx-aggregator/utils"

	"github.com/stretchr/testify/assert"
	"tx-aggregator/model"
)

func TestDivideByDecimals(t *testing.T) {
	tests := []struct {
		value    string
		decimals int
		expected string
	}{
		{"500000000000000000", 18, "0.5"},
		{"1230000000000000000000", 18, "1230"},
		{"100000000", 8, "1"},
		{"1", 18, "0.000000000000000001"},
	}

	for _, tt := range tests {
		result := utils.DivideByDecimals(tt.value, tt.decimals)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMultiplyByDecimals(t *testing.T) {
	tests := []struct {
		value    string
		decimals int
		expected string
	}{
		{"0.5", 18, "500000000000000000"},
		{"1230", 18, "1230000000000000000000"},
		{"1", 8, "100000000"},
		{"0.000000000000000001", 18, "1"},
		{"0", 18, "0"},
	}

	for _, tt := range tests {
		got, err := utils.MultiplyByDecimals(tt.value, tt.decimals)
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, got)
	}
}

func TestMultiplyInvalidFraction(t *testing.T) {
	_, err := utils.MultiplyByDecimals("0.0001", 2) // 4 fractional digits > 2
	assert.Error(t, err)
}

func TestDetectERC20Event(t *testing.T) {
	transferTopic := []string{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"}
	approveTopic := []string{"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"}
	unknownTopic := []string{"0xdeadbeef"}

	txType, addr, val := utils.DetectERC20Event("0xABC", transferTopic, "")
	assert.Equal(t, model.TxTypeTransfer, txType)
	assert.Equal(t, "0xabc", addr)
	assert.Equal(t, "", val)

	txType, addr, val = utils.DetectERC20Event("0xDEF", approveTopic, "0x01")
	assert.Equal(t, model.TxTypeApprove, txType)
	assert.Equal(t, "0xdef", addr)
	assert.Equal(t, "0x01", val)

	txType, addr, val = utils.DetectERC20Event("0xGHI", unknownTopic, "")
	assert.Equal(t, model.TxTypeUnknown, txType)
	assert.Equal(t, "", addr)
	assert.Equal(t, "", val)
}

func TestNormalizeNumericString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		isErr    bool
	}{
		// Valid cases
		{"21000", "21000", false},
		{"0x5208", "21000", false},
		{"  12345  ", "12345", false},
		// Very large numbers (decimal & hex)
		{"20000000000000000000000", "20000000000000000000000", false},
		{"0x11c37937e08000", "5000000000000000", false},
		// Error cases
		{"0xZZZ", "", true},
		{"", "", true},
		{"abc", "", true},
	}

	for _, tt := range tests {
		got, err := utils.NormalizeNumericString(tt.input)
		if tt.isErr {
			assert.Error(t, err, "input=%q expected an error", tt.input)
		} else {
			assert.NoError(t, err, "input=%q should not error", tt.input)
			assert.Equal(t, tt.expected, got, "input=%q", tt.input)
		}
	}
}

func TestParseStringToInt64OrDefault(t *testing.T) {
	assert.Equal(t, int64(21000), utils.ParseStringToInt64OrDefault("21000", 0))
	assert.Equal(t, int64(21000), utils.ParseStringToInt64OrDefault("0x5208", 0))
	assert.Equal(t, int64(0), utils.ParseStringToInt64OrDefault("invalid", 0))
}

func TestParseBlockscoutTimestampToUnix(t *testing.T) {
	ts := "2025-04-16T06:45:02.000000Z"
	unix := utils.ParseBlockscoutTimestampToUnix(ts)
	expected, _ := time.Parse(time.RFC3339Nano, ts)
	assert.Equal(t, expected.Unix(), unix)

	invalid := utils.ParseBlockscoutTimestampToUnix("invalid")
	assert.Equal(t, int64(0), invalid)
}

func TestMergeLogMaps(t *testing.T) {
	src := map[string][]model.BlockscoutLog{
		"tx1": {
			{Address: model.BlockscoutAddressDetails{Hash: "0x1"}},
		},
	}
	dst := map[string][]model.BlockscoutLog{
		"tx1": {
			{Address: model.BlockscoutAddressDetails{Hash: "0x2"}},
		},
		"tx2": {
			{Address: model.BlockscoutAddressDetails{Hash: "0x3"}},
		},
	}

	utils.MergeLogMaps(dst, src)

	assert.Len(t, dst["tx1"], 2)
	assert.Equal(t, "0x2", dst["tx1"][0].Address.Hash)
	assert.Equal(t, "0x1", dst["tx1"][1].Address.Hash)
	assert.Len(t, dst["tx2"], 1)
	assert.Equal(t, "0x3", dst["tx2"][0].Address.Hash)
}

func TestPatchTokenTransactionsWithNormalTxInfo(t *testing.T) {
	normal := model.Transaction{
		Hash:      "0xabc",
		GasLimit:  "21000",
		GasUsed:   "20000",
		GasPrice:  "1000000000",
		Nonce:     "1",
		State:     1,
		BlockHash: "0xblock",
	}
	tokenTxs := []model.Transaction{
		{Hash: "0xabc"},
	}
	result := utils.PatchTokenTransactionsWithNormalTxInfo(tokenTxs, []model.Transaction{normal})
	assert.Equal(t, "21000", result[0].GasLimit)
	assert.Equal(t, "20000", result[0].GasUsed)
	assert.Equal(t, "1000000000", result[0].GasPrice)
	assert.Equal(t, "1", result[0].Nonce)
	assert.Equal(t, 1, result[0].State)
	assert.Equal(t, "0xblock", result[0].BlockHash)
}
