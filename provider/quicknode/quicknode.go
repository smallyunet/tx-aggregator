package quicknode

import (
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/provider"
	"tx-aggregator/types"
	"tx-aggregator/utils"

	"golang.org/x/sync/errgroup"
)

// Ensure we satisfy the Provider interface
var _ provider.Provider = (*QuickNodeProvider)(nil)

// QuickNodeProvider talks to the QuickNode JSON-RPC endpoint.
type QuickNodeProvider struct {
	url      string // *full* QuickNode RPC endpoint – it already embeds the secret key
	chainID  int64  // chain ID implied by the endpoint (e.g. 1 for Ethereum main-net)
	pageSize int    // page size for both normal tx & token transfer queries
	page     int
}

// NewQuickNodeProvider returns a configured provider.
//
// Example:
//
//	qnp := quicknode.NewQuickNodeProvider(
//	         "https://steep-quill-b7b5.quiknode.pro/abcdef123456/", 1, 50)
func NewQuickNodeProvider(endpoint string, chainID int64, pageSize int) *QuickNodeProvider {
	logger.Log.Info().
		Str("endpoint", endpoint).
		Int64("chain_id", chainID).
		Msg("Initialising QuickNodeProvider")
	return &QuickNodeProvider{
		url:      strings.TrimRight(endpoint, "/"),
		chainID:  chainID,
		pageSize: pageSize,
		page:     1,
	}
}

// GetTransactions implements provider.Provider.
// It concurrently fetches on-chain (native) transactions and ERC-20 token transfers
// and converts everything into *types.Transaction*.
func (q *QuickNodeProvider) GetTransactions(params *types.TransactionQueryParams) (*types.TransactionResponse, error) {
	address := params.Address

	var (
		nativeTxs []types.Transaction
		tokenTxs  []types.Transaction
	)

	g := new(errgroup.Group)

	// 1️⃣  native transactions
	g.Go(func() error {
		resp, err := q.getTxByAddress(address, q.page, q.pageSize)
		if err != nil {
			return err
		}
		nativeTxs = q.transformQuickNodeNative(resp, address)
		return nil
	})

	// 2️⃣  token transfers (all contracts)
	g.Go(func() error {
		resp, err := q.getWalletTokenTransfers(address, "", q.page, q.pageSize)
		if err != nil {
			return err
		}
		tokenTxs = q.transformQuickNodeToken(resp, address)
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Merge & return
	all := append(nativeTxs, tokenTxs...)
	return &types.TransactionResponse{
		Result: struct {
			Transactions []types.Transaction `json:"transactions"`
		}{Transactions: all},
		Id: 1,
	}, nil
}

// ---- helpers -------------------------------------------------------------

func (q *QuickNodeProvider) sendRequest(req interface{}, out interface{}) error {
	return utils.DoHttpRequestWithLogging(
		"POST", "quicknode", q.url, req,
		map[string]string{"Content-Type": "application/json"},
		out,
	)
}
