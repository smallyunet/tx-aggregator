package usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"tx-aggregator/config"
	"tx-aggregator/types"

	. "tx-aggregator/usecase"
)

func initTestConfig() {
	cfg := config.Current()
	cfg.ChainNames = map[string]int64{
		"ETH":    1,
		"BSC":    56,
		"CHAINA": 10,
	}
	config.SetCurrentConfig(cfg)
}

func buildResponse(txs []types.Transaction) *types.TransactionResponse {
	resp := &types.TransactionResponse{}
	resp.Result.Transactions = txs
	return resp
}

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

	t.Run("does not remove unrelated native tx", func(t *testing.T) {
		resp := buildResponse([]types.Transaction{
			{Hash: "0x1", CoinType: types.CoinTypeNative},
		})
		FilterNativeShadowTx(resp)
		assert.Len(t, resp.Result.Transactions, 1)
		assert.Equal(t, "0x1", resp.Result.Transactions[0].Hash)
	})
}

func TestFilterTransactionsByInvolvedAddress(t *testing.T) {
	cases := []struct {
		name       string
		txs        []types.Transaction
		queryParam *types.TransactionQueryParams
		wantLen    int
		wantHashes []string
	}{
		{
			name: "match from only",
			txs: []types.Transaction{
				{Hash: "0x1", FromAddress: "0xfrom"},
				{Hash: "0x2", ToAddress: "0xto"},
			},
			queryParam: &types.TransactionQueryParams{Address: "0xfrom", TokenAddress: "0xnotmatch"},
			wantLen:    1,
			wantHashes: []string{"0x1"},
		},
		{
			name: "match to only",
			txs: []types.Transaction{
				{Hash: "0x1", FromAddress: "0xfrom"},
				{Hash: "0x2", ToAddress: "0xto"},
			},
			queryParam: &types.TransactionQueryParams{Address: "0xto", TokenAddress: "0xnotmatch"},
			wantLen:    1,
			wantHashes: []string{"0x2"},
		},
		{
			name: "match token address only",
			txs: []types.Transaction{
				{Hash: "0x3", TokenAddress: "0xtoken"},
			},
			queryParam: &types.TransactionQueryParams{TokenAddress: "0xtoken"},
			wantLen:    1,
			wantHashes: []string{"0x3"},
		},
		{
			name: "no match at all",
			txs: []types.Transaction{
				{Hash: "0x4", FromAddress: "0xabc"},
			},
			queryParam: &types.TransactionQueryParams{Address: "0xnone", TokenAddress: "0xnotmatch"},
			wantLen:    0,
			wantHashes: []string{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := buildResponse(c.txs)
			got := FilterTransactionsByInvolvedAddress(resp, c.queryParam)
			assert.Len(t, got.Result.Transactions, c.wantLen)

			hashes := []string{}
			for _, tx := range got.Result.Transactions {
				hashes = append(hashes, tx.Hash)
			}
			assert.Equal(t, c.wantHashes, hashes)
		})
	}
}

func TestFilterTransactionsByTokenAddress(t *testing.T) {
	resp := buildResponse([]types.Transaction{
		{Hash: "0x1", TokenAddress: "0xtoken", CoinType: types.CoinTypeToken},
		{Hash: "0x2", TokenAddress: "0xtoken", CoinType: types.CoinTypeNative},
	})
	got := FilterTransactionsByTokenAddress(resp, &types.TransactionQueryParams{TokenAddress: "0xtoken"})
	assert.Len(t, got.Result.Transactions, 1)
	assert.Equal(t, "0x1", got.Result.Transactions[0].Hash)
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

	t.Run("single chain ETH", func(t *testing.T) {
		resp := buildResponse([]types.Transaction{
			{ChainID: 1},
			{ChainID: 10},
		})
		got := FilterTransactionsByChainNames(resp, []string{"ETH"})
		assert.Len(t, got.Result.Transactions, 1)
		assert.Equal(t, int64(1), got.Result.Transactions[0].ChainID)
	})

	t.Run("multiple chains ETH & BSC", func(t *testing.T) {
		resp := buildResponse([]types.Transaction{
			{ChainID: 1},
			{ChainID: 56},
			{ChainID: 10},
		})
		got := FilterTransactionsByChainNames(resp, []string{"ETH", "BSC"})
		assert.Len(t, got.Result.Transactions, 2)
	})
}

func TestLimitTransactions(t *testing.T) {
	t.Run("smaller than limit", func(t *testing.T) {
		in := buildResponse([]types.Transaction{{TxIndex: 1}})
		out := LimitTransactions(in, 10)
		assert.Len(t, out.Result.Transactions, 1)
	})

	t.Run("larger than limit", func(t *testing.T) {
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

func TestSortTransactionResponseByHeightAndIndex(t *testing.T) {
	makeResp := func() *types.TransactionResponse {
		return buildResponse([]types.Transaction{
			{Hash: "0xA", Height: 10, TxIndex: 0},
			{Hash: "0xB", Height: 5, TxIndex: 1},
			{Hash: "0xC", Height: 20, TxIndex: 2},
			{Hash: "0xD", Height: 20, TxIndex: 1},
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
}

func TestSetServerChainNames(t *testing.T) {
	initTestConfig()

	resp := buildResponse([]types.Transaction{
		{ChainID: 1},
		{ChainID: 999},
		{ChainID: 10},
	})

	SetServerChainNames(resp)

	assert.Equal(t, []string{"ETH", "", "CHAINA"}, []string{
		resp.Result.Transactions[0].ServerChainName,
		resp.Result.Transactions[1].ServerChainName,
		resp.Result.Transactions[2].ServerChainName,
	})
}
