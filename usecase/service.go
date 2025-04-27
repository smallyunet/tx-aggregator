package usecase

import (
	"tx-aggregator/cache"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/provider"
	"tx-aggregator/types"
)

type Service struct {
	cache    *cache.RedisCache
	provider *provider.MultiProvider
}

func NewService(c *cache.RedisCache, p *provider.MultiProvider) *Service {
	return &Service{
		cache:    c,
		provider: p,
	}
}

func (s *Service) GetTransactions(params *types.TransactionQueryParams) (*types.TransactionResponse, error) {
	logger.Log.Info().
		Str("address", params.Address).
		Str("token_address", params.TokenAddress).
		Interface("chain_names", params.ChainNames).
		Msg("Starting GetTransactions usecase")

	// Step 1: Try reading from cache
	resp, err := s.cache.QueryTxFromCache(params)
	if err == nil && len(resp.Result.Transactions) > 0 {
		logger.Log.Debug().
			Int("transaction_count", len(resp.Result.Transactions)).
			Msg("Transactions loaded from cache")
		return s.postProcess(resp, params), nil
	}

	if err != nil {
		logger.Log.Warn().Err(err).Msg("Error querying transactions from cache")
	} else {
		logger.Log.Debug().Msg("Cache miss: no transactions found")
	}

	// Step 2: Fetch from provider
	logger.Log.Info().Msg("Querying transactions from provider")
	resp, err = s.provider.GetTransactions(params)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Provider query failed")
		code := types.CodeProviderFailed
		return &types.TransactionResponse{
			Code:    code,
			Message: types.GetMessageByCode(code),
		}, err
	}
	logger.Log.Debug().
		Int("fetched_transaction_count", len(resp.Result.Transactions)).
		Msg("Transactions fetched from provider")

	// Step 3: Filter by involved address
	before := len(resp.Result.Transactions)
	resp = FilterTransactionsByInvolvedAddress(resp, params)
	logger.Log.Debug().
		Int("filtered_by_address", len(resp.Result.Transactions)).
		Int("before_filter", before).
		Msg("Filtered transactions by involved address")

	// Step 4: Save to cache
	if err := s.cache.ParseTxAndSaveToCache(resp, params.Address); err != nil {
		logger.Log.Warn().Err(err).Msg("Failed to save fetched transactions to cache")
	} else {
		logger.Log.Debug().Int("cached_transaction_count", len(resp.Result.Transactions)).Msg("Cached transactions successfully")
	}

	// Step 5: Post-process the data
	return s.postProcess(resp, params), nil
}

func (s *Service) postProcess(resp *types.TransactionResponse, params *types.TransactionQueryParams) *types.TransactionResponse {
	// Filter by chain
	before := len(resp.Result.Transactions)
	resp = FilterTransactionsByChainNames(resp, params.ChainNames)
	logger.Log.Debug().
		Int("filtered_by_chain", len(resp.Result.Transactions)).
		Int("before_filter", before).
		Msg("Filtered transactions by chain")

	FilterNativeShadowTx(resp)
	logger.Log.Debug().
		Int("filtered_native_shadow", len(resp.Result.Transactions)).
		Int("before_filter", before).
		Msg("Filtered native shadow transactions")

	// Token or native coin filter
	if params.TokenAddress != "" {
		before = len(resp.Result.Transactions)
		if params.TokenAddress == types.NativeTokenName {
			resp = FilterTransactionsByCoinType(resp, types.CoinTypeNative)
			logger.Log.Debug().
				Int("filtered_native", len(resp.Result.Transactions)).
				Int("before_filter", before).
				Msg("Filtered by native token")
		} else {
			resp = FilterTransactionsByTokenAddress(resp, params)
			logger.Log.Debug().
				Int("filtered_token", len(resp.Result.Transactions)).
				Int("before_filter", before).
				Msg("Filtered by token address")
		}
	}

	// Sort and limit
	SortTransactionResponseByHeightAndIndex(resp, false)
	resp = LimitTransactions(resp, config.AppConfig.Response.Max)
	logger.Log.Debug().
		Int("final_transaction_count", len(resp.Result.Transactions)).
		Msg("Final sorted and limited transaction count")

	// Add chain names to response
	resp = SetServerChainNames(resp)

	// Final response setup
	resp.Code = types.CodeSuccess
	resp.Message = types.GetMessageByCode(types.CodeSuccess)
	return resp
}
