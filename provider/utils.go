package provider

import (
	"strings"
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
