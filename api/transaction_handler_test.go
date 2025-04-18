package api

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"tx-aggregator/model"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockService is a mock implementation of the transaction service interface.
type MockService struct {
	mock.Mock
}

func (m *MockService) GetTransactions(params *model.TransactionQueryParams) (*model.TransactionResponse, error) {
	args := m.Called(params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TransactionResponse), args.Error(1)
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
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Malformed address",
			query:          "address=0x123&token_address=" + validTokenAddr + "&chain_names=ethereum",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Empty query",
			query:          "",
			expectedStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/transactions?"+tt.query, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var body model.TransactionResponse
			err = json.NewDecoder(resp.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, model.CodeInvalidParam, body.Code)
			assert.Equal(t, model.GetMessageByCode(model.CodeInvalidParam), body.Message)
		})
	}
}

// TestGetTransactions_ServiceError tests when service returns an error.
func TestGetTransactions_ServiceError(t *testing.T) {
	mockService := new(MockService)
	app := setupTestApp(mockService)

	paramsMatcher := mock.MatchedBy(func(p *model.TransactionQueryParams) bool {
		return p != nil && strings.EqualFold(p.Address, validAddr)
	})

	mockService.On("GetTransactions", paramsMatcher).Return(nil, errors.New("mock error"))

	req := httptest.NewRequest("GET", "/transactions?address="+validAddr+"&token_address="+validTokenAddr+"&chain_names=eth", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var body model.TransactionResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.Equal(t, model.CodeInternalError, body.Code)
	assert.Equal(t, model.GetMessageByCode(model.CodeInternalError), body.Message)
}

// TestGetTransactions_Success tests successful retrieval from service.
func TestGetTransactions_Success(t *testing.T) {
	mockService := new(MockService)
	app := setupTestApp(mockService)

	expected := &model.TransactionResponse{
		Code:    model.CodeSuccess,
		Message: model.GetMessageByCode(model.CodeSuccess),
	}

	paramsMatcher := mock.MatchedBy(func(p *model.TransactionQueryParams) bool {
		return p != nil && strings.EqualFold(p.Address, validAddr)
	})

	mockService.On("GetTransactions", paramsMatcher).Return(expected, nil)

	req := httptest.NewRequest("GET", "/transactions?address="+validAddr+"&token_address="+validTokenAddr+"&chain_names=ethereum", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.TransactionResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.Equal(t, expected.Code, body.Code)
	assert.Equal(t, expected.Message, body.Message)
}
