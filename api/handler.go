package api

import (
	"github.com/gofiber/fiber/v2"
	"tx-aggregator/cache"
	"tx-aggregator/logger"
	"tx-aggregator/model"
	"tx-aggregator/provider"
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
