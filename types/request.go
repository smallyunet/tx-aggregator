package types

// TransactionRequest defines query parameters from HTTP request
type TransactionRequest struct {
	Chain        string `query:"chain"`        // blockchain name
	Address      string `query:"address"`      // wallet address
	TokenAddress string `query:"tokenAddress"` // optional: filter token transfer
}
