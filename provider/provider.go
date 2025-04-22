package provider

import (
	"context"
	"errors"
	"time"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/types"
)

// Provider defines the interface for transaction data providers.
// Implementations of this interface are responsible for fetching transaction data
// for a given blockchain address from various sources.
type Provider interface {
	GetTransactions(address string) (*types.TransactionResponse, error)
}

// MultiProvider is a composite provider that aggregates results from multiple providers.
// It implements the Provider interface and attempts to fetch transactions from all
// registered providers, combining their results into a single response.
type MultiProvider struct {
	providers []Provider
}

// GetTransactions fetches from every provider concurrently and merges results
func (m *MultiProvider) GetTransactions(address string) (*types.TransactionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.AppConfig.Providers.RequestTimeout)*time.Second)
	defer cancel()

	resCh := make(chan []types.Transaction, len(m.providers))
	errCh := make(chan error, len(m.providers))

	for idx, p := range m.providers {
		go func(i int, prov Provider) {
			start := time.Now()
			resp, err := prov.GetTransactions(address)
			cost := time.Since(start)
			if err != nil {
				logger.Log.Warn().Err(err).Int("provider_index", i).
					Dur("cost", cost).Msg("Provider failed")
				errCh <- err
				return
			}
			logger.Log.Info().Dur("cost", cost).Int("provider_index", i).
				Int("tx_count", len(resp.Result.Transactions)).
				Msg("Provider finished")
			resCh <- resp.Result.Transactions
		}(idx, p)
	}

	var (
		allTxs       []types.Transaction
		successCount int
		failCount    int
	)

	for done := 0; done < len(m.providers); done++ {
		select {
		case txs := <-resCh:
			allTxs = append(allTxs, txs...)
			successCount++
		case err := <-errCh:
			logger.Log.Warn().Err(err).Msg("Provider error")
			failCount++
		case <-ctx.Done():
			return nil, ctx.Err() // global timeout
		}
	}

	close(resCh)
	close(errCh)

	if successCount == 0 && failCount > 0 {
		return nil, errors.New("all providers failed")
	}

	return &types.TransactionResponse{
		Id: 1,
		Result: struct {
			Transactions []types.Transaction `json:"transactions"`
		}{
			Transactions: allTxs,
		},
	}, nil
}

// NewMultiProvider creates a new MultiProvider instance with the given providers.
// It initializes the composite provider that will aggregate results from all provided providers.
func NewMultiProvider(providers ...Provider) *MultiProvider {
	logger.Log.Info().
		Int("provider_count", len(providers)).
		Msg("Initializing new MultiProvider")
	return &MultiProvider{providers: providers}
}
