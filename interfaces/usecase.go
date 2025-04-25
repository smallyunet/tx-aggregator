package interfaces

import "tx-aggregator/types"

// TransactionServiceInterface defines the interface for transaction service
type TransactionServiceInterface interface {
	GetTransactions(params *types.TransactionQueryParams) (*types.TransactionResponse, error)
}
