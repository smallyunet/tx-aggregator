package api

import (
	"fmt"
	"strings"
	"tx-aggregator/cache"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/model"
	"tx-aggregator/provider"
	"tx-aggregator/types"
	"tx-aggregator/usecase"

	"github.com/gofiber/fiber/v2"
)

// Global variables for provider and cache instances
var (
	mulProvider *provider.MultiProvider
	redisCache  *cache.RedisCache
)

// Init initializes the API handlers with provider and cache instances
// p: MultiProvider instance for transaction data
// c: RedisCache instance for caching
func Init(p *provider.MultiProvider, c *cache.RedisCache) {
	mulProvider = p
	redisCache = c
	logger.Log.Info().Msg("API handlers initialized with provider and cache")
}

// GetTransactions handles the HTTP request for fetching transactions
// Returns a JSON response with transaction data or error information
func GetTransactions(ctx *fiber.Ctx) error {
	logger.Log.Info().Msg("Received request for transactions")

	params, err := parseTransactionQueryParams(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to parse transaction query parameters")
		return ctx.Status(fiber.StatusBadRequest).JSON(&model.TransactionResponse{
			Code:    model.CodeInvalidParam,
			Message: model.GetMessageByCode(model.CodeInvalidParam),
		})
	}

	resp, err := handleGetTransactions(ctx, params)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to handle transaction request")
		if resp != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(resp)
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(&model.TransactionResponse{
			Code:    model.CodeInternalError,
			Message: model.GetMessageByCode(model.CodeInternalError),
		})
	}

	logger.Log.Info().Int("transaction_count", len(resp.Result.Transactions)).Msg("Successfully processed transaction request")
	return ctx.JSON(resp)
}

// parseTransactionQueryParams parses and validates query parameters from the request
// Returns TransactionQueryParams struct and error if validation fails
func parseTransactionQueryParams(ctx *fiber.Ctx) (*types.TransactionQueryParams, error) {
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

	params := &types.TransactionQueryParams{
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
func handleGetTransactions(ctx *fiber.Ctx, params *types.TransactionQueryParams) (*model.TransactionResponse, error) {
	logger.Log.Info().
		Interface("params", params).
		Msg("Processing transaction request")

	// Try cache first
	resp, err := redisCache.QueryTxFromCache(params)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to query transactions from cache")
	} else if len(resp.Result.Transactions) > 0 {
		logger.Log.Info().
			Int("cached_transactions", len(resp.Result.Transactions)).
			Msg("Successfully retrieved transactions from cache")
		resp.Code = model.CodeSuccess
		resp.Message = model.GetMessageByCode(model.CodeSuccess)
		return resp, nil
	}

	logger.Log.Info().Msg("Cache miss, fetching transactions from provider")
	// Call provider if not in cache
	rawResp, err := mulProvider.GetTransactions(params.Address)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to get transactions from provider")
		code := model.CodeProviderFailed
		return &model.TransactionResponse{
			Code:    code,
			Message: model.GetMessageByCode(code),
		}, err
	}

	// Apply filter
	rawResp = usecase.FilterTransactionsByAddress(rawResp, params.Address)
	logger.Log.Info().
		Int("filtered_transactions", len(rawResp.Result.Transactions)).
		Msg("Filtered transactions by address")

	if err := redisCache.ParseTxAndSaveToCache(rawResp); err != nil {
		logger.Log.Error().Err(err).Msg("Failed to save transactions to cache")
		code := model.CodeInternalError
		return &model.TransactionResponse{
			Code:    code,
			Message: model.GetMessageByCode(code),
		}, err
	}

	resp, err = redisCache.QueryTxFromCache(params)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to query transactions from cache after save")
		code := model.CodeInternalError
		return &model.TransactionResponse{
			Code:    code,
			Message: model.GetMessageByCode(code),
		}, err
	}

	resp = usecase.LimitTransactions(resp, config.AppConfig.Response.Max)
	logger.Log.Info().
		Int("limited_transactions", len(resp.Result.Transactions)).
		Msg("Applied transaction limit")

	resp.Code = model.CodeSuccess
	resp.Message = model.GetMessageByCode(model.CodeSuccess)
	return resp, nil
}
