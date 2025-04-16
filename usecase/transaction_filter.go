package usecase

import (
	"sort"
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

// SortTransactionResponseByHeightAndHash sorts the transactions inside TransactionResponse.
// It first sorts by height (ascending or descending), then by hash in lexicographical order.
func SortTransactionResponseByHeightAndHash(resp *model.TransactionResponse, ascending bool) {
	if resp == nil || len(resp.Result.Transactions) == 0 {
		return
	}

	sort.Slice(resp.Result.Transactions, func(i, j int) bool {
		txI := resp.Result.Transactions[i]
		txJ := resp.Result.Transactions[j]

		if txI.Height == txJ.Height {
			// If height is equal, compare hash lexicographically
			return txI.Hash < txJ.Hash
		}

		if ascending {
			return txI.Height < txJ.Height
		}
		return txI.Height > txJ.Height
	})
}
