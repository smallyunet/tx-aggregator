package provider

import (
	"strconv"
	"strings"
	"time"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// -----------------------------------------------------------------------------
// Utility Functions
// -----------------------------------------------------------------------------

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

// parseStringToInt64OrDefault converts a string to int64, supporting hex with "0x" prefix
// Returns the default value if parsing fails
func parseStringToInt64OrDefault(s string, def int64) int64 {
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

// parseBlockscoutTimestampToUnix parses a timestamp like "2025-04-16T06:45:02.000000Z" into an int64 (Unix epoch)
func parseBlockscoutTimestampToUnix(ts string) int64 {
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		logger.Log.Warn().
			Err(err).
			Str("timestamp", ts).
			Msg("Failed to parse Tantin timestamp, returning 0")
		return 0
	}
	return parsed.Unix()
}
