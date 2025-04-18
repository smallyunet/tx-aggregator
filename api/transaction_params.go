package api

import (
	"fmt"
	"strings"
	"tx-aggregator/internal/chainmeta"

	"github.com/gofiber/fiber/v2"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/model"
	"tx-aggregator/utils"
)

// parseTransactionQueryParams parses and validates query parameters from the HTTP request context.
// Returns TransactionQueryParams struct and error if validation fails.
func parseTransactionQueryParams(ctx *fiber.Ctx) (*model.TransactionQueryParams, error) {
	address := utils.GetInsensitiveQuery(ctx, "address")
	if address == "" {
		return nil, fmt.Errorf("address parameter is required")
	} else if !utils.IsValidEthereumAddress(address) {
		return nil, fmt.Errorf("invalid address: %s", address)
	}

	// Parse chain names
	chainNamesParam := utils.GetInsensitiveQuery(ctx, "chainName")
	var chainNames []string
	if chainNamesParam == "" {
		// Use all chains
		for name := range config.AppConfig.ChainNames {
			chainNames = append(chainNames, name)
		}
	} else {
		// Validate specified chain names
		logger.Log.Debug().Str("chain_names", chainNamesParam).Msg("Validating specified chain names")
		names := strings.Split(chainNamesParam, ",")
		var unknownChainNames []string

		for _, name := range names {
			name = strings.ToUpper(strings.TrimSpace(name))
			if _, err := chainmeta.ChainIDByName(name); err == nil {
				chainNames = append(chainNames, name)
			} else {
				unknownChainNames = append(unknownChainNames, name)
			}
		}

		if len(unknownChainNames) > 0 {
			return nil, fmt.Errorf("unknown chain names: %s", strings.Join(unknownChainNames, ", "))
		}
	}

	// Parse token address
	tokenAddress := strings.ToLower(utils.GetInsensitiveQuery(ctx, "tokenAddress"))
	if tokenAddress != "" &&
		!utils.IsValidEthereumAddress(tokenAddress) &&
		tokenAddress != model.NativeTokenName {
		return nil, fmt.Errorf("invalid token address: %s", tokenAddress)
	}

	params := &model.TransactionQueryParams{
		Address:      strings.ToLower(address),
		TokenAddress: tokenAddress,
		ChainNames:   chainNames,
	}

	logger.Log.Debug().
		Str("address", params.Address).
		Str("token_address", params.TokenAddress).
		Interface("chain_names", params.ChainNames).
		Msg("Parsed transaction query parameters")

	return params, nil
}
