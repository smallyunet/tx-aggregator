package api

import (
	"fmt"
	"sort"
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

	// Parse and validate chain names
	rawChainNames := utils.GetInsensitiveQuery(ctx, "chainName")
	validChainNames, err := parseAndValidateChainNames(rawChainNames)
	if err != nil {
		return nil, err
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
		ChainNames:   validChainNames,
	}

	logger.Log.Debug().
		Str("address", params.Address).
		Str("token_address", params.TokenAddress).
		Interface("chain_names", params.ChainNames).
		Msg("Parsed transaction query parameters")

	return params, nil
}

// parseAndValidateChainNames validates and normalizes chain names from the input string.
func parseAndValidateChainNames(rawChainNames string) ([]string, error) {
	var validChainNames []string

	if rawChainNames == "" {
		// No input provided, return all available chain names
		for name := range config.AppConfig.ChainNames {
			validChainNames = append(validChainNames, name)
		}
	} else {
		logger.Log.Debug().Str("chain_names", rawChainNames).Msg("Validating specified chain names")
		inputChainNames := strings.Split(rawChainNames, ",")
		var invalidChainNames []string

		for _, name := range inputChainNames {
			normalized := strings.ToUpper(strings.TrimSpace(name))
			if _, err := chainmeta.ChainIDByName(normalized); err == nil {
				validChainNames = append(validChainNames, normalized)
			} else {
				invalidChainNames = append(invalidChainNames, normalized)
			}
		}

		if len(invalidChainNames) > 0 {
			return nil, fmt.Errorf("unknown chain names: %s", strings.Join(invalidChainNames, ", "))
		}
	}

	// Ensure deterministic order
	sort.Strings(validChainNames)
	return validChainNames, nil
}
