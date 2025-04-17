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

// parseTransactionQueryParams parses and validates query parameters from the request
// Returns TransactionQueryParams struct and error if validation fails
func parseTransactionQueryParams(ctx *fiber.Ctx) (*model.TransactionQueryParams, error) {
	address := ctx.Query("address")
	if address == "" {
		return nil, fmt.Errorf("address parameter is required")
	}

	// Get chainNames from query, e.g., "eth,bsc"
	chainNamesParam := ctx.Query("chainName")
	var chainIDs []int64

	if chainNamesParam == "" {
		logger.Log.Debug().Msg("No chain names specified, using all configured blockchains")
		// Use all blockchains if not specified
		for _, id := range config.AppConfig.ChainIDs {
			chainIDs = append(chainIDs, id)
		}
	} else {
		logger.Log.Debug().Str("chain_names", chainNamesParam).Msg("Processing specified chain names")
		// Split by comma and map to chain IDs
		chainNames := strings.Split(chainNamesParam, ",")
		for _, name := range chainNames {
			name = strings.TrimSpace(name)
			if id := config.ChainIDByName(name); id != 0 {
				chainIDs = append(chainIDs, id)
			}
		}
	}

	params := &model.TransactionQueryParams{
		Address:      strings.ToLower(address),
		TokenAddress: strings.ToLower(ctx.Query("tokenAddress")),
		ChainIDs:     chainIDs,
	}

	logger.Log.Debug().
		Str("address", params.Address).
		Str("token_address", params.TokenAddress).
		Interface("chain_ids", params.ChainIDs).
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

	// Check if any transactions were found
	if params.TokenAddress != "" {
		// Filter by token address if provided
		resp = usecase.FilterTransactionsByTokenAddress(resp, params)
		logger.Log.Info().
			Int("filtered_transactions_by_token", len(resp.Result.Transactions)).
			Msg("Filtered transactions by token address")
	}

	// ✅ Sort and limit transactions regardless of source
	usecase.SortTransactionResponseByHeightAndIndex(resp, true)
	logger.Log.Info().
		Int("sorted_transactions", len(resp.Result.Transactions)).
		Msg("Sorted transactions by height and index")

	resp = usecase.LimitTransactions(resp, config.AppConfig.Response.Max)
	logger.Log.Info().
		Int("limited_transactions", len(resp.Result.Transactions)).
		Msg("Applied transaction limit")

	resp.Code = model.CodeSuccess
	resp.Message = model.GetMessageByCode(model.CodeSuccess)
	return resp, nil
}
