package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"tx-aggregator/types"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockService is a mock implementation of the transaction service interface.
type MockService struct {
	mock.Mock
}

func (m *MockService) GetTransactions(params *types.TransactionQueryParams) (*types.TransactionResponse, error) {
	args := m.Called(params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TransactionResponse), args.Error(1)
}

// setupTestApp initializes Fiber app and registers the handler for testing.
func setupTestApp(service *MockService) *fiber.App {
	app := fiber.New()
	handler := NewTransactionHandler(service)
	app.Get("/transactions", handler.GetTransactions)
	return app
}

const (
	validAddr      = "0x1111111111111111111111111111111111111111"
	validTokenAddr = "0x2222222222222222222222222222222222222222"
)

// TestNewTransactionHandler verifies that handler is constructed properly.
func TestNewTransactionHandler(t *testing.T) {
	mockService := new(MockService)
	handler := NewTransactionHandler(mockService)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
}

// TestGetTransactions_InvalidParams tests missing or malformed query parameters.
func TestGetTransactions_InvalidParams(t *testing.T) {
	mockService := new(MockService)
	app := setupTestApp(mockService)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{
			name:           "Missing address parameter",
			query:          "token_address=" + validTokenAddr + "&chain_names=ethereum",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Malformed address",
			query:          "address=0x123&token_address=" + validTokenAddr + "&chain_names=ethereum",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Empty query",
			query:          "",
			expectedStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/transactions?"+tt.query, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var body types.TransactionResponse
			err = json.NewDecoder(resp.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, types.CodeInvalidParam, body.Code)
			assert.Equal(t, types.GetMessageByCode(types.CodeInvalidParam), body.Message)
		})
	}
}

// TestGetTransactions_ServiceError tests when service returns an error.
func TestGetTransactions_ServiceError(t *testing.T) {
	mockService := new(MockService)
	app := setupTestApp(mockService)

	paramsMatcher := mock.MatchedBy(func(p *types.TransactionQueryParams) bool {
		return p != nil && strings.EqualFold(p.Address, validAddr)
	})

	mockService.On("GetTransactions", paramsMatcher).Return(nil, errors.New("mock error"))

	req := httptest.NewRequest("GET", "/transactions?address="+validAddr+"&token_address="+validTokenAddr+"&chain_names=eth", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body types.TransactionResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.Equal(t, types.CodeInternalError, body.Code)
	assert.Equal(t, types.GetMessageByCode(types.CodeInternalError), body.Message)
}

// TestGetTransactions_TimeoutError tests when service returns context.DeadlineExceeded error.
func TestGetTransactions_TimeoutError(t *testing.T) {
	mockService := new(MockService)
	app := setupTestApp(mockService)

	paramsMatcher := mock.MatchedBy(func(p *types.TransactionQueryParams) bool {
		return p != nil && strings.EqualFold(p.Address, validAddr)
	})

	mockService.On("GetTransactions", paramsMatcher).Return(nil, context.DeadlineExceeded)

	req := httptest.NewRequest("GET", "/transactions?address="+validAddr+"&token_address="+validTokenAddr+"&chain_names=ethereum", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body types.TransactionResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.Equal(t, types.CodeProviderFailed, body.Code)
	assert.Equal(t, "Request timed out", body.Message)
}

// TestGetTransactions_ServiceErrorWithResponse tests when service returns error and partial response.
func TestGetTransactions_ServiceErrorWithResponse(t *testing.T) {
	mockService := new(MockService)
	app := setupTestApp(mockService)

	expected := &types.TransactionResponse{
		Code:    types.CodeProviderFailed,
		Message: types.GetMessageByCode(types.CodeProviderFailed),
	}

	paramsMatcher := mock.MatchedBy(func(p *types.TransactionQueryParams) bool {
		return p != nil && strings.EqualFold(p.Address, validAddr)
	})

	mockService.On("GetTransactions", paramsMatcher).Return(expected, errors.New("mock provider failure"))

	req := httptest.NewRequest("GET", "/transactions?address="+validAddr+"&token_address="+validTokenAddr+"&chain_names=ethereum", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body types.TransactionResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.Equal(t, expected.Code, body.Code)
	assert.Equal(t, expected.Message, body.Message)
}

// TestGetTransactions_SuccessWithTransactions tests successful response with transaction data.
func TestGetTransactions_SuccessWithTransactions(t *testing.T) {
	mockService := new(MockService)
	app := setupTestApp(mockService)

	expected := &types.TransactionResponse{
		Code:    types.CodeSuccess,
		Message: types.GetMessageByCode(types.CodeSuccess),
	}
	expected.Result.Transactions = []types.Transaction{
		{
			Hash:        "0xabc123",
			FromAddress: validAddr,
			ToAddress:   validTokenAddr,
			Amount:      "1000",
		},
		{
			Hash:        "0xdef456",
			FromAddress: validAddr,
			ToAddress:   validTokenAddr,
			Amount:      "2000",
		},
	}

	paramsMatcher := mock.MatchedBy(func(p *types.TransactionQueryParams) bool {
		return p != nil && strings.EqualFold(p.Address, validAddr)
	})

	mockService.On("GetTransactions", paramsMatcher).Return(expected, nil)

	req := httptest.NewRequest("GET", "/transactions?address="+validAddr+"&token_address="+validTokenAddr+"&chain_names=ethereum", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body types.TransactionResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.Equal(t, expected.Code, body.Code)
	assert.Equal(t, expected.Message, body.Message)
	assert.Len(t, body.Result.Transactions, 2)
	assert.Equal(t, "0xabc123", body.Result.Transactions[0].Hash)
	assert.Equal(t, "0xdef456", body.Result.Transactions[1].Hash)
}
