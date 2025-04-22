package model

// TransactionType defines the source of the transaction
type TransactionType string

type Transaction struct {
	ServerChainName string `json:"serverChainName"`
	ChainID         int64  `json:"chainId"`
	TokenID         int64  `json:"tokenId"`
	State           int    `json:"state"`
	Height          int64  `json:"height"`
	Hash            string `json:"hash"`
	TxIndex         int64  `json:"txIndex"`
	BlockHash       string `json:"blockHash"`
	FromAddress     string `json:"fromAddress"`
	ToAddress       string `json:"toAddress"`
	TokenAddress    string `json:"tokenAddress"`
	Balance         string `json:"balance"`
	Amount          string `json:"amount"`
	GasUsed         string `json:"gasUsed"`
	GasLimit        string `json:"gasLimit"`
	GasPrice        string `json:"gasPrice"`
	Nonce           string `json:"nonce"`

	// 0: transfer, 1: approve
	Type int `json:"type"`

	// 1: native, 2: token
	CoinType         int    `json:"coinType"`
	TokenDisplayName string `json:"tokenDisplayName"`
	Decimals         int64  `json:"decimals"`

	CreatedTime  int64 `json:"createdTime"`
	ModifiedTime int64 `json:"modifiedTime"`

	// 0: transIn, 1: transOut
	TranType    int    `json:"tranType"`
	ApproveShow string `json:"approveShow"`
	IconURL     string `json:"iconUrl"`
}

type TransactionResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  struct {
		Transactions []Transaction `json:"transactions"`
	} `json:"result"`
	Id int `json:"id"`
}
