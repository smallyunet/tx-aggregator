// File: usecase/usecase_test.go
// Unit-tests for helper functions used by the transaction-aggregator.
//
// Each test uses table-driven style (sub-tests with t.Run) so new cases can be
// appended easily.  All comments are in English as requested.

package usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"tx-aggregator/config"
	"tx-aggregator/types"

	. "tx-aggregator/usecase" // import helpers under test
)

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// initTestConfig injects a deterministic map[chainName]chainID for the tests.
func initTestConfig() {
	cfg := config.Current()
	cfg.ChainNames = map[string]int64{
		"ETH":    1,
		"BSC":    56,
		"CHAINA": 10,
	}
	config.SetCurrentConfig(cfg)
}

// buildResponse is a tiny factory so test cases stay concise.
func buildResponse(txs []types.Transaction) *types.TransactionResponse {
	resp := &types.TransactionResponse{}
	resp.Result.Transactions = txs
	return resp
}

// -----------------------------------------------------------------------------
// FilterNativeShadowTx
// -----------------------------------------------------------------------------

func TestFilterNativeShadowTx(t *testing.T) {
	t.Run("removes duplicate native-shadow pair", func(t *testing.T) {
		resp := buildResponse([]types.Transaction{
			{Hash: "0x1", CoinType: types.CoinTypeNative},
			{Hash: "0x1", CoinType: types.CoinTypeToken}, // shadow
			{Hash: "0x2", CoinType: types.CoinTypeNative},
		})

		FilterNativeShadowTx(resp)
		assert.Len(t, resp.Result.Transactions, 2)
		assert.Equal(t, []string{"0x1", "0x2"}, []string{
			resp.Result.Transactions[0].Hash,
			resp.Result.Transactions[1].Hash,
		})
	})

	t.Run("noop on empty slice", func(t *testing.T) {
		resp := buildResponse(nil)
		FilterNativeShadowTx(resp)
		assert.Empty(t, resp.Result.Transactions)
	})
}

// -----------------------------------------------------------------------------
// LimitTransactions
// -----------------------------------------------------------------------------

func TestLimitTransactions(t *testing.T) {
	t.Run("smaller than limit → unchanged", func(t *testing.T) {
		in := buildResponse([]types.Transaction{{TxIndex: 1}})
		out := LimitTransactions(in, 10)
		assert.Len(t, out.Result.Transactions, 1)
	})

	t.Run("larger than limit → head trimmed", func(t *testing.T) {
		var txs []types.Transaction
		for i := int64(1); i <= 8; i++ {
			txs = append(txs, types.Transaction{TxIndex: i})
		}
		out := LimitTransactions(buildResponse(txs), 5)
		assert.Len(t, out.Result.Transactions, 5)
		assert.Equal(t, int64(1), out.Result.Transactions[0].TxIndex)
		assert.Equal(t, int64(5), out.Result.Transactions[4].TxIndex)
	})
}

// -----------------------------------------------------------------------------
// SortTransactionResponseByHeightAndIndex
// -----------------------------------------------------------------------------

func TestSortTransactionResponseByHeightAndIndex(t *testing.T) {
	// Shared fixture – re-created for each sub-test so we always start unsorted.
	makeResp := func() *types.TransactionResponse {
		return buildResponse([]types.Transaction{
			// different heights   ----------------------------------------------
			{Hash: "0xA", Height: 10, TxIndex: 0},
			{Hash: "0xB", Height: 5, TxIndex: 1},
			// same height, diff index ------------------------------------------
			{Hash: "0xC", Height: 20, TxIndex: 2},
			{Hash: "0xD", Height: 20, TxIndex: 1},
			// same height+index, same from → compare nonce ---------------------
			{Hash: "0xE", Height: 30, TxIndex: 0, FromAddress: "0xdead", Nonce: "8"},
			{Hash: "0xF", Height: 30, TxIndex: 0, FromAddress: "0xdead", Nonce: "3"},
		})
	}

	t.Run("ascending", func(t *testing.T) {
		resp := makeResp()
		SortTransactionResponseByHeightAndIndex(resp, true)
		want := []string{"0xB", "0xA", "0xD", "0xC", "0xF", "0xE"}
		for i, h := range want {
			assert.Equal(t, h, resp.Result.Transactions[i].Hash)
		}
	})

	t.Run("descending", func(t *testing.T) {
		resp := makeResp()
		SortTransactionResponseByHeightAndIndex(resp, false)
		want := []string{"0xE", "0xF", "0xC", "0xD", "0xA", "0xB"}
		for i, h := range want {
			assert.Equal(t, h, resp.Result.Transactions[i].Hash)
		}
	})

	t.Run("nil response should not panic", func(t *testing.T) {
		SortTransactionResponseByHeightAndIndex(nil, true)
	})
}

// -----------------------------------------------------------------------------
// SetServerChainNames
// -----------------------------------------------------------------------------

func TestSetServerChainNames(t *testing.T) {
	initTestConfig()

	resp := buildResponse([]types.Transaction{
		{ChainID: 1},   // ETH
		{ChainID: 999}, // unmapped
		{ChainID: 10},  // CHAINA
	})

	SetServerChainNames(resp)

	assert.Equal(t, []string{"ETH", "", "CHAINA"}, []string{
		resp.Result.Transactions[0].ServerChainName,
		resp.Result.Transactions[1].ServerChainName,
		resp.Result.Transactions[2].ServerChainName,
	})
}

// -----------------------------------------------------------------------------
// FilterTransactions* helpers
// -----------------------------------------------------------------------------

func TestFilterTransactionsByInvolvedAddress(t *testing.T) {
	resp := buildResponse([]types.Transaction{
		{Hash: "0x1", FromAddress: "0xfrom"},
		{Hash: "0x2", ToAddress: "0xto"},
	})

	cases := []struct {
		name    string
		addr    string
		wantLen int
	}{
		{"match from", "0xfrom", 1},
		{"match to", "0xto", 1},
		{"no match", "0xnone", 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FilterTransactionsByInvolvedAddress(resp, &types.TransactionQueryParams{Address: c.addr})
			assert.Len(t, got.Result.Transactions, c.wantLen)
		})
	}
}

func TestFilterTransactionsByTokenAddress(t *testing.T) {
	resp := buildResponse([]types.Transaction{
		{Hash: "0x1", TokenAddress: "0xtoken"},
	})

	got := FilterTransactionsByTokenAddress(resp, &types.TransactionQueryParams{TokenAddress: "0xtoken"})
	assert.Len(t, got.Result.Transactions, 1)

	got = FilterTransactionsByTokenAddress(resp, &types.TransactionQueryParams{TokenAddress: "0xnone"})
	assert.Empty(t, got.Result.Transactions)
}

func TestFilterTransactionsByCoinType(t *testing.T) {
	resp := buildResponse([]types.Transaction{
		{Hash: "0x1", CoinType: types.CoinTypeToken},
		{Hash: "0x2", CoinType: types.CoinTypeNative},
	})

	got := FilterTransactionsByCoinType(resp, types.CoinTypeToken)
	assert.Len(t, got.Result.Transactions, 1)
	assert.Equal(t, "0x1", got.Result.Transactions[0].Hash)
}

func TestFilterTransactionsByChainNames(t *testing.T) {
	initTestConfig()

	resp := buildResponse([]types.Transaction{
		{ChainID: 1},
		{ChainID: 10},
	})

	got := FilterTransactionsByChainNames(resp, []string{"eth"})
	assert.Len(t, got.Result.Transactions, 1)
	assert.Equal(t, int64(1), got.Result.Transactions[0].ChainID)
}
