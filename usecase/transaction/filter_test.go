package transaction

import (
	"testing"
	"tx-aggregator/config"
	"tx-aggregator/model"

	"github.com/stretchr/testify/assert"
)

func TestFilterTransactionsByInvolvedAddress(t *testing.T) {
	resp := &model.TransactionResponse{}
	resp.Result.Transactions = []model.Transaction{
		{FromAddress: "0xabc", ToAddress: "0xdef", TokenAddress: "0x123"},
		{FromAddress: "0xdef", ToAddress: "0xAbC", TokenAddress: "0x456"},
		{FromAddress: "0xghi", ToAddress: "0xjkl", TokenAddress: "0XABC"},
		{FromAddress: "0xzzz", ToAddress: "0xyyy", TokenAddress: "0xtoken"},
	}
	params := &model.TransactionQueryParams{Address: "0xAbC", TokenAddress: "0xabc"}
	filtered := FilterTransactionsByInvolvedAddress(resp, params)
	assert.Len(t, filtered.Result.Transactions, 3)
}

func TestFilterTransactionsByTokenAddress(t *testing.T) {
	resp := &model.TransactionResponse{}
	resp.Result.Transactions = []model.Transaction{
		{TokenAddress: "0x111"},
		{TokenAddress: "0x222"},
		{TokenAddress: "0X111"},
	}
	params := &model.TransactionQueryParams{TokenAddress: "0x111"}
	filtered := FilterTransactionsByTokenAddress(resp, params)
	assert.Len(t, filtered.Result.Transactions, 2)
}

func TestFilterTransactionsByCoinType(t *testing.T) {
	resp := &model.TransactionResponse{}
	resp.Result.Transactions = []model.Transaction{
		{CoinType: 1},
		{CoinType: 2},
		{CoinType: 1},
	}
	filtered := FilterTransactionsByCoinType(resp, 1)
	assert.Len(t, filtered.Result.Transactions, 2)
}

func TestFilterTransactionsByChainNames(t *testing.T) {
	// No chainNames specified: should return original
	resp := &model.TransactionResponse{}
	resp.Result.Transactions = []model.Transaction{{ChainID: 1}, {ChainID: 2}}
	filtered := FilterTransactionsByChainNames(resp, []string{})
	assert.Len(t, filtered.Result.Transactions, 2)

	// Specified but no match: should return empty
	resp = &model.TransactionResponse{}
	resp.Result.Transactions = []model.Transaction{{ChainID: 3}, {ChainID: 4}}
	filtered = FilterTransactionsByChainNames(resp, []string{"foo"})
	assert.Len(t, filtered.Result.Transactions, 0)

	// ChainID=0 matches default id from ChainIDByName("any") => id=0
	resp = &model.TransactionResponse{}
	resp.Result.Transactions = []model.Transaction{{ChainID: 0}, {ChainID: 1}}
	filtered = FilterTransactionsByChainNames(resp, []string{"any"})
	assert.Len(t, filtered.Result.Transactions, 1)
	assert.Equal(t, int64(0), filtered.Result.Transactions[0].ChainID)
}

func TestSortTransactionResponseByHeightAndIndex(t *testing.T) {
	resp := &model.TransactionResponse{}
	resp.Result.Transactions = []model.Transaction{
		{Height: 2, TxIndex: 1},
		{Height: 1, TxIndex: 3},
		{Height: 1, TxIndex: 2},
	}
	// Ascending
	SortTransactionResponseByHeightAndIndex(resp, true)
	asc := resp.Result.Transactions
	assert.Equal(t, int64(1), asc[0].Height)
	assert.Equal(t, int64(2), asc[0].TxIndex)
	assert.Equal(t, int64(1), asc[1].Height)
	assert.Equal(t, int64(3), asc[1].TxIndex)
	assert.Equal(t, int64(2), asc[2].Height)

	// Descending
	SortTransactionResponseByHeightAndIndex(resp, false)
	desc := resp.Result.Transactions
	assert.Equal(t, int64(2), desc[0].Height)
	assert.Equal(t, int64(1), desc[1].Height)
	assert.Equal(t, int64(3), desc[1].TxIndex)
	assert.Equal(t, int64(1), desc[2].Height)
	assert.Equal(t, int64(2), desc[2].TxIndex)
}

func TestLimitTransactions(t *testing.T) {
	resp := &model.TransactionResponse{}
	resp.Result.Transactions = []model.Transaction{
		{TxIndex: 1},
		{TxIndex: 2},
		{TxIndex: 3},
	}
	limited := LimitTransactions(resp, 2)
	assert.Len(t, limited.Result.Transactions, 2)
	assert.Equal(t, int64(1), limited.Result.Transactions[0].TxIndex)
	assert.Equal(t, int64(2), limited.Result.Transactions[1].TxIndex)
}

func TestSetServerChainNames(t *testing.T) {
	// Seed mapping
	config.AppConfig.ChainNames = map[string]int64{"chaina": 10}
	resp := &model.TransactionResponse{}
	resp.Result.Transactions = []model.Transaction{
		{ChainID: 10},
		{ChainID: 99},
	}
	SetServerChainNames(resp)
	assert.Equal(t, "CHAINA", resp.Result.Transactions[0].ServerChainName)
	assert.Equal(t, "", resp.Result.Transactions[1].ServerChainName)
}
