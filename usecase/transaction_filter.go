package usecase

import (
	"sort"
	"strings"
	"tx-aggregator/model"
)

// FilterTransactionsByInvolvedAddress filters transactions to only include those where the address
// is either the sender or the receiver.
func FilterTransactionsByInvolvedAddress(resp *model.TransactionResponse, params *model.TransactionQueryParams) *model.TransactionResponse {
	filtered := make([]model.Transaction, 0, len(resp.Result.Transactions))
	addrLower := strings.ToLower(params.Address)
	tokenAddrLower := strings.ToLower(params.TokenAddress)

	for _, tx := range resp.Result.Transactions {
		if strings.ToLower(tx.FromAddress) == addrLower || strings.ToLower(tx.ToAddress) == addrLower || strings.ToLower(tx.TokenAddress) == tokenAddrLower {
			filtered = append(filtered, tx)
		}
	}

	resp.Result.Transactions = filtered
	return resp
}

// FilterTransactionsByTokenAddress filters transactions to only include those with the specified token address.
func FilterTransactionsByTokenAddress(resp *model.TransactionResponse, params *model.TransactionQueryParams) *model.TransactionResponse {
	filtered := make([]model.Transaction, 0, len(resp.Result.Transactions))
	tokenAddrLower := strings.ToLower(params.TokenAddress)

	for _, tx := range resp.Result.Transactions {
		if strings.ToLower(tx.TokenAddress) == tokenAddrLower {
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

// SortTransactionResponseByHeightAndIndex sorts transactions by block height and txIndex.
// If heights are the same, it sorts by txIndex in ascending order.
func SortTransactionResponseByHeightAndIndex(resp *model.TransactionResponse, ascending bool) {
	if resp == nil || len(resp.Result.Transactions) == 0 {
		return
	}

	sort.Slice(resp.Result.Transactions, func(i, j int) bool {
		txI := resp.Result.Transactions[i]
		txJ := resp.Result.Transactions[j]

		if txI.Height == txJ.Height {
			if ascending {
				return txI.TxIndex < txJ.TxIndex
			}
			return txI.TxIndex > txJ.TxIndex
		}

		if ascending {
			return txI.Height < txJ.Height
		}
		return txI.Height > txJ.Height
	})
}
