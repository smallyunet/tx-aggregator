package transaction

import "tx-aggregator/model"

// ServiceInterface defines the interface for transaction service
type ServiceInterface interface {
	GetTransactions(params *model.TransactionQueryParams) (*model.TransactionResponse, error)
}
