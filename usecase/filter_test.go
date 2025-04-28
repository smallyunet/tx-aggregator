package usecase

import (
	"testing"
	"tx-aggregator/config"
	"tx-aggregator/types"

	"github.com/stretchr/testify/assert"
)

func TestFilterNativeShadowTx(t *testing.T) {
	resp := &types.TransactionResponse{}
	resp.Result.Transactions = []types.Transaction{
		{Hash: "0x1", CoinType: types.CoinTypeNative},
		{Hash: "0x1", CoinType: types.CoinTypeToken},
		{Hash: "0x2", CoinType: types.CoinTypeNative},
		{Hash: "0x3", CoinType: types.CoinTypeToken},
	}

	FilterNativeShadowTx(resp)
	assert.Len(t, resp.Result.Transactions, 3)
	hashes := make(map[string]bool)
	for _, tx := range resp.Result.Transactions {
		hashes[tx.Hash] = true
	}
	assert.True(t, hashes["0x1"])
	assert.True(t, hashes["0x2"])
	assert.True(t, hashes["0x3"])
}

func TestLimitTransactions_SmallList(t *testing.T) {
	resp := &types.TransactionResponse{}
	resp.Result.Transactions = []types.Transaction{
		{TxIndex: 1},
	}
	limited := LimitTransactions(resp, 5) // limit > actual size
	assert.Len(t, limited.Result.Transactions, 1)
	assert.Equal(t, int64(1), limited.Result.Transactions[0].TxIndex)
}

func TestSortTransactionResponseByHeightAndIndex_Empty(t *testing.T) {
	var resp *types.TransactionResponse
	SortTransactionResponseByHeightAndIndex(resp, true) // Should not panic
	resp = &types.TransactionResponse{}
	SortTransactionResponseByHeightAndIndex(resp, false) // Should not panic
	assert.Empty(t, resp.Result.Transactions)
}

func TestSetServerChainNames_Unmapped(t *testing.T) {
	config.AppConfig.ChainNames = map[string]int64{"chaina": 10}
	resp := &types.TransactionResponse{}
	resp.Result.Transactions = []types.Transaction{
		{ChainID: 999},
	}
	SetServerChainNames(resp)
	assert.Equal(t, "", resp.Result.Transactions[0].ServerChainName)
}

func TestFilterTransactionsByInvolvedAddress_Empty(t *testing.T) {
	resp := &types.TransactionResponse{}
	params := &types.TransactionQueryParams{Address: "0xabc"}
	filtered := FilterTransactionsByInvolvedAddress(resp, params)
	assert.Len(t, filtered.Result.Transactions, 0)
}

func TestFilterTransactionsByTokenAddress_Empty(t *testing.T) {
	resp := &types.TransactionResponse{}
	params := &types.TransactionQueryParams{TokenAddress: "0xabc"}
	filtered := FilterTransactionsByTokenAddress(resp, params)
	assert.Len(t, filtered.Result.Transactions, 0)
}

func TestFilterTransactionsByCoinType_Empty(t *testing.T) {
	resp := &types.TransactionResponse{}
	filtered := FilterTransactionsByCoinType(resp, types.CoinTypeToken)
	assert.Len(t, filtered.Result.Transactions, 0)
}

func TestFilterTransactionsByChainNames_InvalidName(t *testing.T) {
	resp := &types.TransactionResponse{}
	resp.Result.Transactions = []types.Transaction{{ChainID: 100}}
	filtered := FilterTransactionsByChainNames(resp, []string{"nonexistent"})
	assert.Len(t, filtered.Result.Transactions, 0)
}
