package transaction

import (
	"sort"
	"strings"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

// FilterTransactionsByInvolvedAddress filters transactions to only include those where the address
// is either the sender or the receiver.
func FilterTransactionsByInvolvedAddress(resp *types.TransactionResponse, params *types.TransactionQueryParams) *types.TransactionResponse {
	filtered := make([]types.Transaction, 0, len(resp.Result.Transactions))
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
func FilterTransactionsByTokenAddress(resp *types.TransactionResponse, params *types.TransactionQueryParams) *types.TransactionResponse {
	filtered := make([]types.Transaction, 0, len(resp.Result.Transactions))
	tokenAddrLower := strings.ToLower(params.TokenAddress)

	for _, tx := range resp.Result.Transactions {
		if strings.ToLower(tx.TokenAddress) == tokenAddrLower {
			filtered = append(filtered, tx)
		}
	}

	resp.Result.Transactions = filtered
	return resp
}

// FilterTransactionsByCoinType filters transactions to only include those with the specified coin type.
func FilterTransactionsByCoinType(resp *types.TransactionResponse, coinType int) *types.TransactionResponse {
	filtered := make([]types.Transaction, 0, len(resp.Result.Transactions))

	for _, tx := range resp.Result.Transactions {
		if tx.CoinType == coinType {
			filtered = append(filtered, tx)
		}
	}

	resp.Result.Transactions = filtered
	return resp
}

// FilterTransactionsByChainNames filters transactions to only include those with the specified chain IDs.
func FilterTransactionsByChainNames(resp *types.TransactionResponse, chainNames []string) *types.TransactionResponse {
	if len(chainNames) == 0 {
		return resp
	}

	// Use a set for fast lookup
	chainIDSet := make(map[int64]struct{}, len(chainNames))
	for _, name := range chainNames {
		id, _ := utils.ChainIDByName(name)
		chainIDSet[id] = struct{}{}
	}

	filtered := make([]types.Transaction, 0, len(resp.Result.Transactions))
	for _, tx := range resp.Result.Transactions {
		if _, ok := chainIDSet[tx.ChainID]; ok {
			filtered = append(filtered, tx)
		}
	}

	resp.Result.Transactions = filtered
	return resp
}

// SortTransactionResponseByHeightAndIndex sorts transactions by block height and txIndex.
// If heights are the same, it sorts by txIndex in ascending order.
func SortTransactionResponseByHeightAndIndex(resp *types.TransactionResponse, ascending bool) {
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

// LimitTransactions limits the number of transactions to a maximum count.
func LimitTransactions(resp *types.TransactionResponse, max int) *types.TransactionResponse {
	txs := resp.Result.Transactions
	if len(txs) > max {
		resp.Result.Transactions = txs[:max]
	}
	return resp
}

// SetServerChainNames sets the ServerChainName field for each transaction
// based on the chain ID using the configured chain name mappings.
func SetServerChainNames(resp *types.TransactionResponse) *types.TransactionResponse {
	for i, tx := range resp.Result.Transactions {
		name, _ := utils.ChainNameByID(tx.ChainID)
		resp.Result.Transactions[i].ServerChainName = name
	}
	return resp
}
