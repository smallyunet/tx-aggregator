package api

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"strings"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/model"
	"tx-aggregator/usecase"
)

// getInsensitiveQuery retrieves the query parameter value regardless of the case
func getInsensitiveQuery(ctx *fiber.Ctx, key string) string {
	queryParams := ctx.Queries() // map[string]string
	for k, v := range queryParams {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}

// parseTransactionQueryParams parses and validates query parameters from the request
// Returns TransactionQueryParams struct and error if validation fails
func parseTransactionQueryParams(ctx *fiber.Ctx) (*model.TransactionQueryParams, error) {
	address := getInsensitiveQuery(ctx, "address")
	if address == "" {
		return nil, fmt.Errorf("address parameter is required")
	} else if !isValidEthereumAddress(address) {
		return nil, fmt.Errorf("invalid address: %s", address)
	}

	// Get chainNames from query, e.g., "eth,bsc"
	chainNamesParam := getInsensitiveQuery(ctx, "chainName")
	var chainNames []string
	if chainNamesParam == "" {
		logger.Log.Debug().Msg("No chain names specified, using all configured blockchains")
		// Use all blockchains if not specified
		for name, _ := range config.AppConfig.ChainNames {
			chainNames = append(chainNames, name)
		}
	} else {
		logger.Log.Debug().Str("chain_names", chainNamesParam).Msg("Processing specified chain names")

		// Split by comma and map to chain IDs
		chainsParam := strings.Split(chainNamesParam, ",")
		var unknownChainNames []string

		for _, name := range chainsParam {
			name = strings.TrimSpace(name)
			if _, err := config.ChainIDByName(name); err == nil {
				chainNames = append(chainNames, name)
			} else {
				unknownChainNames = append(unknownChainNames, name)
			}
		}

		// Return error if any chain names are not recognized
		if len(unknownChainNames) > 0 {
			return nil, fmt.Errorf("unknown chain names: %s", strings.Join(unknownChainNames, ", "))
		}
	}

	tokenAddressParam := strings.ToLower(getInsensitiveQuery(ctx, "tokenAddress"))
	if tokenAddressParam != "" && !isValidEthereumAddress(tokenAddressParam) && tokenAddressParam != model.NativeTokenName {
		return nil, fmt.Errorf("invalid token address: %s", tokenAddressParam)
	}

	params := &model.TransactionQueryParams{
		Address:      strings.ToLower(address),
		TokenAddress: tokenAddressParam,
		ChainNames:   chainNames,
	}

	logger.Log.Debug().
		Str("address", params.Address).
		Str("token_address", params.TokenAddress).
		Interface("chain_names", params.ChainNames).
		Msg("Successfully parsed transaction query parameters")

	return params, nil
}

// handleGetTransactions processes the transaction request by checking cache and provider
// Returns TransactionResponse and error if processing fails
func handleGetTransactions(ctx *fiber.Ctx, params *model.TransactionQueryParams) (*model.TransactionResponse, error) {
	logger.Log.Info().
		Interface("params", params).
		Msg("Processing transaction request")

	var resp *model.TransactionResponse
	var err error

	// Try cache first
	resp, err = redisCache.QueryTxFromCache(params)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to query transactions from cache")
	} else if len(resp.Result.Transactions) > 0 {
		logger.Log.Info().
			Int("cached_transactions", len(resp.Result.Transactions)).
			Msg("Successfully retrieved transactions from cache")
	} else {
		logger.Log.Info().Msg("Cache miss, fetching transactions from provider")
		// Call provider if not in cache
		resp, err = mulProvider.GetTransactions(params.Address)
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to get transactions from provider")
			code := model.CodeProviderFailed
			return &model.TransactionResponse{
				Code:    code,
				Message: model.GetMessageByCode(code),
			}, err
		}

		// Apply filter
		resp = usecase.FilterTransactionsByInvolvedAddress(resp, params)
		logger.Log.Info().
			Int("filtered_transactions", len(resp.Result.Transactions)).
			Msg("Filtered transactions by involoved address")

		// Save to cache
		if err := redisCache.ParseTxAndSaveToCache(resp, params.Address); err != nil {
			logger.Log.Error().Err(err).Msg("Failed to save transactions to cache")
			code := model.CodeInternalError
			return &model.TransactionResponse{
				Code:    code,
				Message: model.GetMessageByCode(code),
			}, err
		}
	}

	logger.Log.Info().
		Int("cached_transactions", len(resp.Result.Transactions)).
		Msg("Successfully fetched transactions from provider")

	// Filter transactions by chain ID
	resp = usecase.FilterTransactionsByChainNames(resp, params.ChainNames)
	logger.Log.Info().
		Int("filtered_transactions_by_chain_id", len(resp.Result.Transactions)).
		Msg("Filtered transactions by chain ID")

	// Check if any transactions were found
	if params.TokenAddress != "" {
		if params.TokenAddress != model.NativeTokenName {
			// Filter by token address if provided
			resp = usecase.FilterTransactionsByTokenAddress(resp, params)
			logger.Log.Info().
				Int("filtered_transactions_by_token", len(resp.Result.Transactions)).
				Msg("Filtered transactions by token address")
		} else if params.TokenAddress == model.NativeTokenName {
			resp = usecase.FilterTransactionsByCoinType(resp, model.CoinTypeNative)
			logger.Log.Info().
				Int("filtered_transactions_by_coin_type", len(resp.Result.Transactions)).
				Msg("Filtered transactions by coin type")
		}
	}

	// Sort and limit transactions regardless of source
	usecase.SortTransactionResponseByHeightAndIndex(resp, true)
	logger.Log.Info().
		Int("sorted_transactions", len(resp.Result.Transactions)).
		Msg("Sorted transactions by height and index")

	resp = usecase.LimitTransactions(resp, config.AppConfig.Response.Max)
	logger.Log.Info().
		Int("limited_transactions", len(resp.Result.Transactions)).
		Msg("Applied transaction limit")

	resp = usecase.SetServerChainNames(resp)
	logger.Log.Info().
		Int("server_chain_names_set", len(resp.Result.Transactions)).
		Msg("Set server chain names for transactions")

	resp.Code = model.CodeSuccess
	resp.Message = model.GetMessageByCode(model.CodeSuccess)
	return resp, nil
}
