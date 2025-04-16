package usecase

import (
	"strings"
	"tx-aggregator/model"
)

// FilterTransactionsByAddress filters transactions to only include those where the address
// is either the sender or the receiver.
func FilterTransactionsByAddress(resp *model.TransactionResponse, address string) *model.TransactionResponse {
	filtered := make([]model.Transaction, 0, len(resp.Result.Transactions))
	addrLower := strings.ToLower(address)

	for _, tx := range resp.Result.Transactions {
		if strings.ToLower(tx.FromAddress) == addrLower || strings.ToLower(tx.ToAddress) == addrLower {
			filtered = append(filtered, tx)
		}
	}

	resp.Result.Transactions = filtered
	return resp
}

// LimitTransactions limits the number of transactions to a maximum count.
func LimitTransactions(resp *model.TransactionResponse, max int) *model.TransactionResponse {
	txs := resp.Result.Transactions
	if len(txs) > max {
		resp.Result.Transactions = txs[:max]
	}
	return resp
}
