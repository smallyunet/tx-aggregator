package provider

import (
	"strconv"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

func DetectERC20Type(logs []model.LogEntry) (typ int, tokenAddress string, approveValue string) {
	for _, log := range logs {
		if len(log.Topics) == 0 {
			continue
		}
		topic0 := strings.ToLower(log.Topics[0])
		switch topic0 {
		case "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef":
			return 0, log.Address, "" // transfer
		case "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925":
			return 1, log.Address, log.Data // approve 的 value 就在 data 字段里
		}
	}
	return -1, "", ""
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
