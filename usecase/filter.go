package usecase

import (
	"sort"
	"strconv"
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
		if strings.ToLower(tx.TokenAddress) == tokenAddrLower && tx.CoinType == types.CoinTypeToken {
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

// Deterministic version: adds FromAddress and keeps stability.
func SortTransactionResponseByHeightAndIndex(resp *types.TransactionResponse, ascending bool) {
	if resp == nil || len(resp.Result.Transactions) == 0 {
		return
	}

	sort.SliceStable(resp.Result.Transactions, func(i, j int) bool {
		txI, txJ := resp.Result.Transactions[i], resp.Result.Transactions[j]

		if txI.Height != txJ.Height {
			return ascending == (txI.Height < txJ.Height)
		}
		if txI.TxIndex != txJ.TxIndex {
			return ascending == (txI.TxIndex < txJ.TxIndex)
		}
		if txI.FromAddress == txJ.FromAddress {
			if nI, errI := strconv.ParseUint(txI.Nonce, 10, 64); errI == nil {
				if nJ, errJ := strconv.ParseUint(txJ.Nonce, 10, 64); errJ == nil && nI != nJ {
					return ascending == (nI < nJ)
				}
			}
		}
		if txI.Hash != txJ.Hash {
			return ascending == (txI.Hash < txJ.Hash)
		}
		// Final fallback to FromAddress for deterministic order
		return ascending == (txI.FromAddress < txJ.FromAddress)
	})
}

// LimitTransactions limits the number of transactions to a maximum count.
func LimitTransactions(resp *types.TransactionResponse, max int64) *types.TransactionResponse {
	txs := resp.Result.Transactions
	if int64(len(txs)) > max {
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

// FilterNativeShadowTx removes the redundant native (coinType == 1) “shadow”
// / transaction that accompanies an ERC-20 transfer (coinType == 2) with the
// same hash. The function rewrites resp.Result.Transactions in place.
func FilterNativeShadowTx(resp *types.TransactionResponse) {
	if resp == nil || len(resp.Result.Transactions) == 0 {
		return // nothing to filter
	}

	// Pass 1: collect the hashes of every ERC-20 transfer.
	tokenTxHashes := make(map[string]struct{}, len(resp.Result.Transactions))
	for _, tx := range resp.Result.Transactions {
		if tx.CoinType == 2 {
			tokenTxHashes[tx.Hash] = struct{}{}
		}
	}

	// Pass 2: copy transactions we want to keep into the same slice:
	//   • all token transfers
	//   • any native transfer that is *not* a zero-value shadow of a token transfer
	keep := resp.Result.Transactions[:0] // reuse underlying memory
	for _, tx := range resp.Result.Transactions {
		if tx.CoinType == 1 {
			if _, paired := tokenTxHashes[tx.Hash]; paired {
				// Skip the shadow native transaction.
				continue
			}
		}
		keep = append(keep, tx)
	}

	resp.Result.Transactions = keep
}
