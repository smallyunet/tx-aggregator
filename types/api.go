package types

// TransactionQueryParams represents the parameters for querying transactions
type TransactionQueryParams struct {
	Address      string
	TokenAddress string
	ChainNames   []string
}
