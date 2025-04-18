package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidEthereumAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		valid   bool
	}{
		{
			name:    "valid lowercase address",
			address: "0x0123456789abcdef0123456789abcdef01234567",
			valid:   true,
		},
		{
			name:    "valid mixed-case address",
			address: "0xAbCDeF1234567890aBCdef1234567890ABcDEf12",
			valid:   true,
		},
		{
			name:    "invalid: missing 0x prefix",
			address: "abcdef0123456789abcdef0123456789abcdef01",
			valid:   false,
		},
		{
			name:    "invalid: too short",
			address: "0x1234567890abcdef",
			valid:   false,
		},
		{
			name:    "invalid: too long",
			address: "0x0123456789abcdef0123456789abcdef0123456789",
			valid:   false,
		},
		{
			name:    "invalid: contains non-hex chars",
			address: "0x01234G6789abcdef0123456789abcdef01234567",
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEthereumAddress(tt.address)
			assert.Equal(t, tt.valid, result)
		})
	}
}
