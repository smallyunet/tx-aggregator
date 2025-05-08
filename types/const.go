package types

const (
	ConfigFolderPath = "configfiles"
)

// CoinType represents the type of cryptocurrency
const (
	// CoinTypeNative represents native cryptocurrency (e.g., ETH, BNB)
	CoinTypeNative = 1
	// CoinTypeToken represents ERC20/ERC721 tokens
	CoinTypeToken = 2
	// CoinTypeInternal represents internal transactions (e.g., contract interactions)
	CoinTypeInternal = 3

	// NativeTokenName is the name for native tokens
	NativeTokenName = "native"
)

// TxType represents the type of transaction
const (
	TxTypeUnknown = 0 // native token transfer also as transfer
	// TxTypeTransfer represents a standard transfer transaction
	TxTypeTransfer = 0
	// TxTypeApprove represents an approval transaction for token spending
	TxTypeApprove = 1
	// TxTypeInternal represents an internal transaction (e.g., contract interaction)
	TxTypeInternal = 2
)

// TransType represents the direction of transaction
const (
	// TransTypeIn represents incoming transactions
	TransTypeIn = 0
	// TransTypeOut represents outgoing transactions
	TransTypeOut = 1
)

// NativeDefaultDecimals represents the default number of decimal places
// for native cryptocurrencies (e.g., 18 decimals for ETH)
const NativeDefaultDecimals = 18

const (
	// TxStateSuccess represents a successful transaction
	TxStateSuccess = 1
	// TxStateFail represents a failed transaction
	TxStateFail = 0
)
