package utils

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
	"tx-aggregator/logger"
	"tx-aggregator/model"
	"unicode"
)

// DetectERC20Event checks if the (address, topics, data) indicate
// an ERC-20 Transfer or Approval event.
//
// Returns:
//   - txType: model.TxTypeTransfer (0), model.TxTypeApprove (1), or -1 if unrecognized
//   - tokenAddress: the address of the ERC-20 token (lowercased)
//   - approveValue: hex-encoded amount (only non-empty if it's an Approval event)
func DetectERC20Event(
	contractAddress string,
	topics []string,
	data string,
) (txType int, tokenAddress string, approveValue string) {

	// Full 32-byte event signatures:
	const transferSig = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	const approveSig = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"

	if len(topics) == 0 {
		return model.TxTypeUnknown, "", ""
	}

	// Convert to lower for matching
	topic0 := strings.ToLower(topics[0])
	addrLower := strings.ToLower(contractAddress)

	switch topic0 {
	case transferSig:
		// This is an ERC-20 Transfer event
		return model.TxTypeTransfer, addrLower, ""

	case approveSig:
		// This is an ERC-20 Approval event
		// The amount is typically in the log's data field
		return model.TxTypeApprove, addrLower, data

	default:
		// Not recognized
		return model.TxTypeUnknown, "", ""
	}
}

// Within wherever you loop over logs in a transaction:
func DetectERC20TypeForAnkr(logs []model.AnkrLogEntry) (typ int, tokenAddress, approveValue string) {
	for _, log := range logs {
		txType, tAddr, appVal := DetectERC20Event(log.Address, log.Topics, log.Data)
		if txType != model.TxTypeUnknown {
			// As soon as you detect a recognized event, you can return it.
			// Or, if you want to keep searching for multiple, you can adapt logic.
			return txType, tAddr, appVal
		}
	}
	return model.TxTypeUnknown, "", ""
}

// ParseStringToInt64OrDefault converts a string to int64, supporting hex with "0x" prefix
// Returns the default value if parsing fails
func ParseStringToInt64OrDefault(s string, def int64) int64 {
	var val int64
	var err error

	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		val, err = strconv.ParseInt(s[2:], 16, 64)
	} else {
		val, err = strconv.ParseInt(s, 10, 64)
	}

	if err != nil {
		logger.Log.Warn().
			Err(err).
			Str("input", s).
			Int64("default", def).
			Msg("Failed to parse string to int64, using default value")
		return def
	}
	return val
}

// ParseBlockscoutTimestampToUnix parses a timestamp like "2025-04-16T06:45:02.000000Z" into an int64 (Unix epoch)
func ParseBlockscoutTimestampToUnix(ts string) int64 {
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		logger.Log.Warn().
			Err(err).
			Str("timestamp", ts).
			Msg("Failed to parse Blockscout timestamp, returning 0")
		return 0
	}
	return parsed.Unix()
}

// MergeLogMaps appends logs from src into dst (keyed by tx hash).
// Duplicate logs are allowed; add deduplication here if required.
func MergeLogMaps(dst, src map[string][]model.BlockscoutLog) {
	for hash, logs := range src {
		dst[hash] = append(dst[hash], logs...)
	}
}

// NormalizeNumericString converts a numeric string—either
//   - hexadecimal (prefix "0x"/"0X"), or
//   - decimal
//
// into a canonical decimal string with no leading zeros.
//
// Examples
//
//	"0x5208"         -> "21000"
//	"21000"          -> "21000"
//	"  12345  "      -> "12345"
func NormalizeNumericString(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", errors.New("empty input string")
	}

	var z big.Int
	switch {
	case strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X"):
		// Hexadecimal (strip "0x")
		if _, ok := z.SetString(input[2:], 16); !ok {
			return "", fmt.Errorf("invalid hex string %q", input)
		}
	default:
		// Decimal
		if _, ok := z.SetString(input, 10); !ok {
			return "", fmt.Errorf("invalid decimal string %q", input)
		}
	}

	return z.String(), nil
}

// PatchTokenTransactionsWithNormalTxInfo updates token transactions with gas-related fields
// by looking up matching tx hash from the normal transactions.
func PatchTokenTransactionsWithNormalTxInfo(
	tokenTxs []model.Transaction,
	normalTxs []model.Transaction,
) []model.Transaction {
	// Build a lookup map from normal transactions
	txMap := make(map[string]model.Transaction, len(normalTxs))
	for _, tx := range normalTxs {
		txMap[tx.Hash] = tx
	}

	// Patch token transactions
	for i, tokenTx := range tokenTxs {
		if normal, ok := txMap[tokenTx.Hash]; ok {
			tokenTxs[i].GasLimit = normal.GasLimit
			tokenTxs[i].GasUsed = normal.GasUsed
			tokenTxs[i].GasPrice = normal.GasPrice
			tokenTxs[i].Nonce = normal.Nonce
			tokenTxs[i].State = normal.State
			tokenTxs[i].BlockHash = normal.BlockHash
		}
	}
	return tokenTxs
}

// DivideByDecimals converts an integer string to a decimal string by shifting the dot
// `value`   – integer in base‑10 (no sign, no “0x” prefix)
// `decimals`– how many decimals the original integer assumed
// Example: DivideByDecimals("1", 18) == "0.000000000000000001"
func DivideByDecimals(value string, decimals int) string {
	// Remove leading zeros to simplify later logic.
	value = strings.TrimLeft(value, "0")
	if value == "" {
		value = "0"
	}
	if decimals == 0 {
		return value
	}

	// If the number of digits ≤ decimals, we need to left‑pad with zeros:
	//     1 / 10¹⁸  -> "000...001" (19 chars) -> "0.000...001"
	if len(value) <= decimals {
		padding := strings.Repeat("0", decimals-len(value)+1)
		value = padding + value
	}

	// Insert decimal point.
	dot := len(value) - decimals
	res := value[:dot] + "." + value[dot:]

	// Trim any trailing zeros and a possible trailing dot.
	res = strings.TrimRight(res, "0")
	res = strings.TrimRight(res, ".")

	return res
}

// MultiplyByDecimals converts a decimal‑string to its integer representation
// by shifting the dot `decimals` places to the right.
//
//	value    — decimal in base‑10 (no sign, may contain one “.”)
//	decimals — how many decimals the *target* integer should assume
//
// Example: MultiplyByDecimals("0.1", 18) == "100000000000000000"
func MultiplyByDecimals(value string, decimals int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("empty input string")
	}

	// Split into integer‑part and fractional‑part.
	parts := strings.SplitN(value, ".", 2)
	intPart := parts[0]
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}

	// Validate that both parts contain only digits.
	isDigits := func(s string) bool {
		for _, r := range s {
			if !unicode.IsDigit(r) {
				return false
			}
		}
		return true
	}
	if !isDigits(intPart) || !isDigits(fracPart) {
		return "", fmt.Errorf("invalid numeric string: %q", value)
	}

	// Too many fractional digits → cannot represent exactly.
	if len(fracPart) > decimals {
		return "", fmt.Errorf(
			"%q has %d fractional digits, exceeds token decimals %d",
			value, len(fracPart), decimals,
		)
	}

	// Strip leading zeros on the integer part
	intPart = strings.TrimLeft(intPart, "0")
	if intPart == "" {
		intPart = "0"
	}

	// Pad the *right* side with zeros until we reach desired precision.
	padded := intPart + fracPart + strings.Repeat("0", decimals-len(fracPart))

	// Remove any residual leading zeros (but keep one if the number is 0).
	padded = strings.TrimLeft(padded, "0")
	if padded == "" {
		padded = "0"
	}
	return padded, nil
}
