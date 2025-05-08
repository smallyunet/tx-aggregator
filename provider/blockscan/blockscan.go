package blockscan

import (
	"golang.org/x/sync/errgroup"
	"tx-aggregator/logger"
	"tx-aggregator/provider"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

// Make sure we satisfy the common Provider interface.
var _ provider.Provider = (*BlockscanProvider)(nil)

// BlockscanProvider fetches data from a BscScan / Etherscan compatible REST API.
type BlockscanProvider struct {
	chainID int64
	cfg     types.BlockscanConfig
}

// NewBlockscanProvider constructs a provider for one chain / one base-URL.
func NewBlockscanProvider(chainID int64, cfg types.BlockscanConfig) *BlockscanProvider {
	logger.Log.Info().
		Str("url", cfg.URL).
		Str("chain", cfg.ChainName).
		Msg("Initializing BlockscanProvider")
	return &BlockscanProvider{
		chainID: chainID,
		cfg:     cfg,
	}
}

// -----------------------------------------------------------------------------
// Public entry – fan-out, merge and return a single TransactionResponse
// -----------------------------------------------------------------------------

func (p *BlockscanProvider) GetTransactions(params *types.TransactionQueryParams) (*types.TransactionResponse, error) {
	address := params.Address

	logger.Log.Info().
		Str("provider", p.cfg.ChainName).
		Str("address", address).
		Msg("Fetching transactions from Blockscan")

	var (
		normalTxs   []types.Transaction
		internalTxs []types.Transaction
		tokenTxs    []types.Transaction
	)

	g := new(errgroup.Group)

	// 1. Normal transactions (txlist)
	g.Go(func() error {
		resp, err := p.fetchNormalTx(address)
		if err != nil {
			return err
		}
		normalTxs = p.transformNormalTx(resp, address)
		return nil
	})

	// 2. Token transfers (tokentx)
	g.Go(func() error {
		resp, err := p.fetchTokenTx(address)
		if err != nil {
			return err
		}
		tokenTxs = p.transformTokenTx(resp, address)
		return nil
	})

	// 3. Internal transactions (txlistinternal)
	// TODO: temporarily disabled due to API issues
	//g.Go(func() error {
	//	resp, err := p.fetchInternalTx(address)
	//	if err != nil {
	//		return err
	//	}
	//	internalTxs = p.transformInternalTx(resp, address)
	//	return nil
	//})

	// Wait for all three API calls
	if err := g.Wait(); err != nil {
		logger.Log.Error().Err(err).Msg("Blockscan fetch failed")
		return nil, err
	}

	// Patch gas info into token transfers
	tokenTxs = utils.PatchTokenTransactionsWithNormalTxInfo(tokenTxs, normalTxs)

	all := append(normalTxs, tokenTxs...)
	all = append(all, internalTxs...)

	logger.Log.Info().
		Str("provider", p.cfg.ChainName).
		Int("normal", len(normalTxs)).
		Int("token", len(tokenTxs)).
		Int("internal", len(internalTxs)).
		Int("total", len(all)).
		Msg("Blockscan provider finished")

	return &types.TransactionResponse{
		Result: struct {
			Transactions []types.Transaction `json:"transactions"`
		}{Transactions: all},
	}, nil
}
