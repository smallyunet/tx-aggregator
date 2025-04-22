package provider

import (
	"errors"
	"sort"
	"testing"
	"time"
	"tx-aggregator/config"
	"tx-aggregator/types"

	"github.com/stretchr/testify/assert"
)

// mockProvider is a test implementation of the Provider interface
type mockProvider struct {
	transactions []types.Transaction
	err          error
	delay        time.Duration
}

func (m *mockProvider) GetTransactions(address string) (*types.TransactionResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.err != nil {
		return nil, m.err
	}
	return &types.TransactionResponse{
		Code:    0,
		Message: "ok",
		Id:      1,
		Result: struct {
			Transactions []types.Transaction `json:"transactions"`
		}{
			Transactions: m.transactions,
		},
	}, nil
}

func TestMultiProvider_AllSuccess(t *testing.T) {
	config.AppConfig.Providers.RequestTimeout = 3 // seconds

	p1 := &mockProvider{transactions: []types.Transaction{{Hash: "0xabc"}}}
	p2 := &mockProvider{transactions: []types.Transaction{{Hash: "0xdef"}}}
	mp := NewMultiProvider(p1, p2)

	resp, err := mp.GetTransactions("0x123")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(resp.Result.Transactions))
	sort.Slice(resp.Result.Transactions, func(i, j int) bool {
		return resp.Result.Transactions[i].Hash < resp.Result.Transactions[j].Hash
	})
	assert.Equal(t, "0xabc", resp.Result.Transactions[0].Hash)
	assert.Equal(t, "0xdef", resp.Result.Transactions[1].Hash)
}

func TestMultiProvider_SomeFail(t *testing.T) {
	config.AppConfig.Providers.RequestTimeout = 3 // seconds

	p1 := &mockProvider{
		transactions: []types.Transaction{{Hash: "0xaaa"}},
	}
	p2 := &mockProvider{
		err: errors.New("provider failed"),
	}
	mp := NewMultiProvider(p1, p2)

	resp, err := mp.GetTransactions("0x123")

	assert.NoError(t, err, "should still succeed if at least one provider works")
	assert.NotNil(t, resp)

	var hashes []string
	for _, tx := range resp.Result.Transactions {
		hashes = append(hashes, tx.Hash)
	}
	assert.Contains(t, hashes, "0xaaa")
}

func TestMultiProvider_AllFail(t *testing.T) {
	config.AppConfig.Providers.RequestTimeout = 3

	p1 := &mockProvider{err: errors.New("failed")}
	p2 := &mockProvider{err: errors.New("also failed")}
	mp := NewMultiProvider(p1, p2)

	resp, err := mp.GetTransactions("0x123")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestMultiProvider_Timeout(t *testing.T) {
	config.AppConfig.Providers.RequestTimeout = 1 // seconds

	p1 := &mockProvider{
		transactions: []types.Transaction{{Hash: "0xdelayed"}},
		delay:        2 * time.Second, // exceeds timeout
	}
	mp := NewMultiProvider(p1)

	resp, err := mp.GetTransactions("0x123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Nil(t, resp)
}
