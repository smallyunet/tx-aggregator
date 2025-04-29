package provider

import (
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"tx-aggregator/config"
	"tx-aggregator/types"
)

// mockProvider is a fake provider used for testing
type mockProvider struct {
	transactions []types.Transaction
	err          error
	delay        time.Duration
}

func (m *mockProvider) GetTransactions(params *types.TransactionQueryParams) (*types.TransactionResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.err != nil {
		return nil, m.err
	}
	return &types.TransactionResponse{
		Code:    0,
		Message: "ok",
		Result: struct {
			Transactions []types.Transaction `json:"transactions"`
		}{
			Transactions: m.transactions,
		},
	}, nil
}

// prepareTestMultiProvider creates a MultiProvider with fake providers and config
func prepareTestMultiProvider(providers map[string]Provider, chainMap map[string]string) *MultiProvider {
	config.AppConfig.Providers.RequestTimeout = 3
	config.AppConfig.Providers.ChainProviders = chainMap
	return NewMultiProvider(providers)
}

func TestMultiProvider_AllSuccess(t *testing.T) {
	p1 := &mockProvider{transactions: []types.Transaction{{Hash: "0xabc"}}}
	p2 := &mockProvider{transactions: []types.Transaction{{Hash: "0xdef"}}}

	providerMap := map[string]Provider{
		"p1": p1,
		"p2": p2,
	}
	chainMap := map[string]string{
		"eth": "p1",
		"bsc": "p2",
	}

	mp := prepareTestMultiProvider(providerMap, chainMap)

	params := &types.TransactionQueryParams{
		ChainNames: []string{"eth", "bsc"},
	}
	resp, err := mp.GetTransactions(params)
	assert.NoError(t, err)
	assert.Len(t, resp.Result.Transactions, 2)

	// Sort by hash for deterministic order
	sort.Slice(resp.Result.Transactions, func(i, j int) bool {
		return resp.Result.Transactions[i].Hash < resp.Result.Transactions[j].Hash
	})

	assert.Equal(t, "0xabc", resp.Result.Transactions[0].Hash)
	assert.Equal(t, "0xdef", resp.Result.Transactions[1].Hash)
}

func TestMultiProvider_SomeFail(t *testing.T) {
	p1 := &mockProvider{transactions: []types.Transaction{{Hash: "0xaaa"}}}
	p2 := &mockProvider{err: errors.New("provider failed")}

	mp := prepareTestMultiProvider(
		map[string]Provider{"p1": p1, "p2": p2},
		map[string]string{"eth": "p1", "bsc": "p2"},
	)

	params := &types.TransactionQueryParams{ChainNames: []string{"eth", "bsc"}}
	resp, err := mp.GetTransactions(params)

	assert.NoError(t, err)
	assert.NotNil(t, resp)

	var hashes []string
	for _, tx := range resp.Result.Transactions {
		hashes = append(hashes, tx.Hash)
	}
	assert.Contains(t, hashes, "0xaaa")
}

func TestMultiProvider_AllFail(t *testing.T) {
	p1 := &mockProvider{err: errors.New("failed")}
	p2 := &mockProvider{err: errors.New("also failed")}

	mp := prepareTestMultiProvider(
		map[string]Provider{"p1": p1, "p2": p2},
		map[string]string{"eth": "p1", "bsc": "p2"},
	)

	params := &types.TransactionQueryParams{ChainNames: []string{"eth", "bsc"}}
	resp, err := mp.GetTransactions(params)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestMultiProvider_DelayedButWithinTimeout(t *testing.T) {
	config.AppConfig.Providers.RequestTimeout = 3 // 3 seconds timeout

	p1 := &mockProvider{
		transactions: []types.Transaction{{Hash: "0xdelayed"}},
		delay:        2 * time.Second, // still within timeout
	}

	mp := prepareTestMultiProvider(
		map[string]Provider{"p1": p1},
		map[string]string{"eth": "p1"},
	)

	params := &types.TransactionQueryParams{ChainNames: []string{"eth"}}
	resp, err := mp.GetTransactions(params)

	// ✅ Expect success
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Result.Transactions, 1)
	assert.Equal(t, "0xdelayed", resp.Result.Transactions[0].Hash)
}
