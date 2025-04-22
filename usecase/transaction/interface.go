package transaction

import "tx-aggregator/types"

// ServiceInterface defines the interface for transaction service
type ServiceInterface interface {
	GetTransactions(params *types.TransactionQueryParams) (*types.TransactionResponse, error)
}
