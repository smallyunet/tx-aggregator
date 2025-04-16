package provider

import (
	"sync"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// Provider defines the interface for transaction data providers.
// Implementations of this interface are responsible for fetching transaction data
// for a given blockchain address from various sources.
type Provider interface {
	GetTransactions(address string) (*model.TransactionResponse, error)
}

// MultiProvider is a composite provider that aggregates results from multiple providers.
// It implements the Provider interface and attempts to fetch transactions from all
// registered providers, combining their results into a single response.
type MultiProvider struct {
	providers []Provider
}

// GetTransactions concurrently fetches transactions from all registered providers.
// It aggregates successful results and returns a combined response.
// If all providers fail, it returns the last encountered error.
func (m *MultiProvider) GetTransactions(address string) (*model.TransactionResponse, error) {
	logger.Log.Info().Str("address", address).Msg("Fetching transactions from multiple providers")

	var wg sync.WaitGroup // WaitGroup to wait for all goroutines
	var mu sync.Mutex     // Mutex to protect shared resources
	var allTransactions []model.Transaction
	var lastErr error // Used to record the last error in case all fail

	for i, p := range m.providers {
		wg.Add(1)
		// Concurrently execute each provider
		go func(i int, provider Provider) {
			defer wg.Done()

			logger.Log.Debug().Int("provider_index", i).Msg("Attempting to fetch from provider")

			res, err := provider.GetTransactions(address)
			if err != nil {
				logger.Log.Warn().
					Err(err).
					Int("provider_index", i).
					Msg("Provider failed")
				// Store the last error in case all providers fail
				mu.Lock()
				lastErr = err
				mu.Unlock()
				return
			}

			// If transactions are successfully retrieved, add them to the aggregate slice
			if res != nil && res.Result.Transactions != nil {
				transactionCount := len(res.Result.Transactions)
				logger.Log.Info().
					Int("provider_index", i).
					Int("transaction_count", transactionCount).
					Msg("Successfully fetched transactions from provider")

				mu.Lock()
				allTransactions = append(allTransactions, res.Result.Transactions...)
				mu.Unlock()
			}
		}(i, p)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	logger.Log.Info().
		Int("total_transactions", len(allTransactions)).
		Str("address", address).
		Msg("Successfully aggregated transactions from providers")

	// If no transactions were fetched and lastErr is not nil, it means all providers failed
	if len(allTransactions) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return &model.TransactionResponse{
		Id: 1, // You can customize this ID
		Result: struct {
			Transactions []model.Transaction `json:"transactions"`
		}{
			Transactions: allTransactions,
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
