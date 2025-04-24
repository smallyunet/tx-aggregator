// Package provider implements a pluggable, fan-out / fan-in layer that merges
// the results of several concrete data sources (Ankr, multiple Blockscout
// instances, …) into a single TransactionResponse.
//
// Configuration-driven routing
// ----------------------------
//
//  1. **Provider registry** – built in main.go, keyed by a *providerKey* string.
//     registry := map[string]Provider{
//     "ankr"            : ankrProvider,
//     "blockscout_ttx"  : bsTtxProvider,
//     "blockscout_abc"  : bsAbcProvider,
//     }
//
//  2. **Chain → providerKey map** – parsed from YAML:
//
//     providers:
//     chain_providers:
//     ETH : ankr
//     BSC : ankr
//     TTX : blockscout_ttx
//     ABC : blockscout_abc
//
//  3. When the API call includes `params.ChainNames`, only the providers mapped
//     to those chain names are invoked; if the slice is empty we invoke *all*
//     providers that appear in `chain_providers`.
package provider

import (
	"context"
	"errors"
	"strings"
	"time"

	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/types"
)

// Provider is the interface every concrete data source must satisfy.
type Provider interface {
	GetTransactions(params *types.TransactionQueryParams) (*types.TransactionResponse, error)
}

// MultiProvider dispatches a single request to several Providers concurrently
// and merges their results.
type MultiProvider struct {
	providers      map[string]Provider // providerKey -> concrete provider
	chainProviders map[string]string   // chainName   -> providerKey (from YAML)
}

// NewMultiProvider builds a MultiProvider from an already-initialised registry.
func NewMultiProvider(registry map[string]Provider) *MultiProvider {
	return &MultiProvider{
		providers:      registry,
		chainProviders: config.AppConfig.Providers.ChainProviders, // YAML-driven
	}
}

// GetTransactions decides which concrete providers to call, fans out the
// requests, waits for all of them (or a global timeout), merges the
// Transaction slices, and returns a single response.
func (m *MultiProvider) GetTransactions(params *types.TransactionQueryParams) (*types.TransactionResponse, error) {
	// ----- 1. Choose providers ------------------------------------------------
	needed := make(map[string]Provider) // providerKey -> Provider

	if len(params.ChainNames) == 0 {
		// Client did not specify chains → use every provider referenced in YAML.
		for _, key := range m.chainProviders {
			if p, ok := m.providers[key]; ok {
				needed[key] = p
			}
		}
	} else {
		// Filter by requested chain names.
		for _, chain := range params.ChainNames {
			chain = strings.ToLower(strings.TrimSpace(chain))
			if key, ok := m.chainProviders[chain]; ok {
				if p, ok2 := m.providers[key]; ok2 {
					needed[key] = p
				} else {
					logger.Log.Warn().
						Str("provider_key", key).
						Msg("Provider key listed in YAML but not registered")
				}
			} else {
				logger.Log.Warn().
					Str("chain_name", chain).
					Msg("No provider mapping for chain")
			}
		}
	}

	if len(needed) == 0 {
		return nil, errors.New("no providers selected for requested chains")
	}

	// ----- 2. Fan-out calls ---------------------------------------------------
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(config.AppConfig.Providers.RequestTimeout)*time.Second,
	)
	defer cancel()

	resCh := make(chan []types.Transaction, len(needed))
	errCh := make(chan error, len(needed))

	idx := 0
	for key, p := range needed {
		go func(i int, prov Provider, name string) {
			start := time.Now()
			resp, err := prov.GetTransactions(params)
			cost := time.Since(start)

			if err != nil {
				logger.Log.Warn().
					Err(err).
					Str("provider", name).
					Dur("cost", cost).
					Msg("Provider failed")
				errCh <- err
				return
			}

			logger.Log.Info().
				Str("provider", name).
				Dur("cost", cost).
				Int("tx_count", len(resp.Result.Transactions)).
				Msg("Provider finished")
			resCh <- resp.Result.Transactions
		}(idx, p, key)
		idx++
	}

	// ----- 3. Collect results -------------------------------------------------
	var (
		allTxs       []types.Transaction
		successCount int
		failCount    int
	)

	for done := 0; done < len(needed); done++ {
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

	if successCount == 0 && failCount > 0 {
		return nil, errors.New("all selected providers failed")
	}

	// ----- 4. Merge & return --------------------------------------------------
	return &types.TransactionResponse{
		Id: 1,
		Result: struct {
			Transactions []types.Transaction `json:"transactions"`
		}{
			Transactions: allTxs,
		},
	}, nil
}
