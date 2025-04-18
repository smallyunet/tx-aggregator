package cache

import (
	"testing"
	"tx-aggregator/model"
)

func TestFormatChainKey(t *testing.T) {
	tests := []struct {
		address   string
		chainName string
		expected  string
	}{
		{"0xABCDEF", "ETH", "0xabcdef-eth"},
		{"0x123456", "bsc", "0x123456-bsc"},
		{"0xABCDEF", "Polygon", "0xabcdef-polygon"},
	}

	for _, tt := range tests {
		result := formatChainKey(tt.address, tt.chainName)
		if result != tt.expected {
			t.Errorf("formatChainKey(%q, %q) = %q; want %q", tt.address, tt.chainName, result, tt.expected)
		}
	}
}

func TestFormatNativeKey(t *testing.T) {
	tests := []struct {
		address   string
		chainName string
		expected  string
	}{
		{"0xABCDEF", "ETH", "0xabcdef-eth-" + model.NativeTokenName},
		{"0x123456", "bsc", "0x123456-bsc-" + model.NativeTokenName},
	}

	for _, tt := range tests {
		result := formatNativeKey(tt.address, tt.chainName)
		if result != tt.expected {
			t.Errorf("formatNativeKey(%q, %q) = %q; want %q", tt.address, tt.chainName, result, tt.expected)
		}
	}
}

func TestFormatTokenKey(t *testing.T) {
	tests := []struct {
		address   string
		chainName string
		tokenAddr string
		expected  string
	}{
		{"0xABCDEF", "ETH", "0xToken1", "0xabcdef-eth-0xtoken1"},
		{"0x123456", "bsc", "0xdeadbeef", "0x123456-bsc-0xdeadbeef"},
	}

	for _, tt := range tests {
		result := formatTokenKey(tt.address, tt.chainName, tt.tokenAddr)
		if result != tt.expected {
			t.Errorf("formatTokenKey(%q, %q, %q) = %q; want %q", tt.address, tt.chainName, tt.tokenAddr, result, tt.expected)
		}
	}
}

func TestFormatTokenSetKey(t *testing.T) {
	tests := []struct {
		address   string
		chainName string
		expected  string
	}{
		{"0xABCDEF", "ETH", "0xabcdef-eth-tokens"},
		{"0x123456", "bsc", "0x123456-bsc-tokens"},
	}

	for _, tt := range tests {
		result := formatTokenSetKey(tt.address, tt.chainName)
		if result != tt.expected {
			t.Errorf("formatTokenSetKey(%q, %q) = %q; want %q", tt.address, tt.chainName, result, tt.expected)
		}
	}
}
