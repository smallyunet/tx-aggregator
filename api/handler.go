package api

import (
	"context"
	"errors"
	"github.com/gofiber/fiber/v2"
	"time"
	"tx-aggregator/interfaces"
	"tx-aggregator/logger"
	"tx-aggregator/types"
)

// TransactionHandler handles HTTP requests related to transaction queries.
type TransactionHandler struct {
	service interfaces.TransactionServiceInterface
}

// NewTransactionHandler initializes a new TransactionHandler with the given service.
func NewTransactionHandler(service interfaces.TransactionServiceInterface) *TransactionHandler {
	return &TransactionHandler{service: service}
}

// GetTransactions handles GET /transactions.
// It parses query parameters, delegates processing to the usecase, and always returns HTTP 200,
// with the actual status represented by a custom code in the JSON body.
func (h *TransactionHandler) GetTransactions(ctx *fiber.Ctx) error {
	start := time.Now()
	logger.Log.Info().Msg("📥 Received /transactions request")

	// Parse and validate query parameters
	params, err := parseTransactionQueryParams(ctx)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("❌ Invalid query parameters")
		return ctx.JSON(&types.TransactionResponse{
			Code:    types.CodeInvalidParam,
			Message: types.GetMessageByCode(types.CodeInvalidParam),
		})
	}

	logger.Log.Info().
		Str("address", params.Address).
		Str("token_address", params.TokenAddress).
		Interface("chain_names", params.ChainNames).
		Msg("✅ Parsed transaction request parameters")

	// Call the usecase/service layer
	resp, err := h.service.GetTransactions(params)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Dur("cost", time.Since(start)).
			Msg("❌ Error while processing transaction request")

		// Handle timeout explicitly
		if errors.Is(err, context.DeadlineExceeded) {
			return ctx.JSON(&types.TransactionResponse{
				Code:    types.CodeProviderFailed, // Or define a CodeTimeout if you prefer
				Message: "Request timed out",
			})
		}

		// Generic internal error
		if resp == nil {
			resp = &types.TransactionResponse{
				Code:    types.CodeInternalError,
				Message: types.GetMessageByCode(types.CodeInternalError),
			}
		}

		// Always return HTTP 200, embed error in response body
		return ctx.JSON(resp)
	}

	// Log and return successful response
	logger.Log.Info().
		Int("tx_count", len(resp.Result.Transactions)).
		Int("code", resp.Code).
		Dur("cost", time.Since(start)).
		Msg("✅ Successfully retrieved transaction data")

	return ctx.JSON(resp)
}
