package api

import (
	"github.com/gofiber/fiber/v2"
	"tx-aggregator/logger"
	"tx-aggregator/model"
	transactionUsecase "tx-aggregator/usecase/transaction"
)

// TransactionHandler handles HTTP requests related to transaction queries.
type TransactionHandler struct {
	service *transactionUsecase.Service
}

// NewTransactionHandler creates a new instance of TransactionHandler with the provided service.
func NewTransactionHandler(service *transactionUsecase.Service) *TransactionHandler {
	return &TransactionHandler{service: service}
}

// GetTransactions handles GET /transactions endpoint.
// It parses query parameters, delegates to the usecase, and returns the result or error.
func (h *TransactionHandler) GetTransactions(ctx *fiber.Ctx) error {
	logger.Log.Info().Msg("Received /transactions request")

	params, err := parseTransactionQueryParams(ctx)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("Invalid query parameters")
		return ctx.Status(fiber.StatusBadRequest).JSON(&model.TransactionResponse{
			Code:    model.CodeInvalidParam,
			Message: model.GetMessageByCode(model.CodeInvalidParam),
		})
	}

	logger.Log.Info().
		Str("address", params.Address).
		Str("token_address", params.TokenAddress).
		Interface("chain_names", params.ChainNames).
		Msg("Parsed transaction request parameters")

	resp, err := h.service.GetTransactions(params)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Usecase returned error during transaction processing")

		// ensure resp is not nil
		if resp == nil {
			resp = &model.TransactionResponse{
				Code:    model.CodeInternalError,
				Message: model.GetMessageByCode(model.CodeInternalError),
			}
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(resp)
	}

	return ctx.JSON(resp)
}
