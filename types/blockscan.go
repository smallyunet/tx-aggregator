package types

// -----------------------------------------------------------------------------
// JSON response structs (minimal fields only)
// -----------------------------------------------------------------------------

type BlockscanNormalTxResp struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Result  []BlockscanTxItem `json:"result"`
}

type BlockscanInternalTxResp struct {
	Status  string                  `json:"status"`
	Message string                  `json:"message"`
	Result  []BlockscanInternalItem `json:"result"`
}

type BlockscanTokenTxResp struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Result  []BlockscanTokenTxItem `json:"result"`
}

type BlockscanTxItem struct {
	BlockNumber      string `json:"blockNumber"`
	TimeStamp        string `json:"timeStamp"`
	Hash             string `json:"hash"`
	Nonce            string `json:"nonce"`
	BlockHash        string `json:"blockHash"`
	TransactionIndex string `json:"transactionIndex"`
	From             string `json:"from"`
	To               string `json:"to"`
	Value            string `json:"value"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	GasUsed          string `json:"gasUsed"`
	IsError          string `json:"isError"`          // 0 / 1
	TxReceiptStatus  string `json:"txreceipt_status"` // 0 / 1
}

type BlockscanInternalItem struct {
	BlockNumber string `json:"blockNumber"`
	TimeStamp   string `json:"timeStamp"`
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	Gas         string `json:"gas"`
	GasUsed     string `json:"gasUsed"`
	IsError     string `json:"isError"`
}

type BlockscanTokenTxItem struct {
	BlockNumber      string `json:"blockNumber"`
	TimeStamp        string `json:"timeStamp"`
	Hash             string `json:"hash"`
	BlockHash        string `json:"blockHash"`
	From             string `json:"from"`
	To               string `json:"to"`
	ContractAddress  string `json:"contractAddress"`
	Value            string `json:"value"`
	TokenName        string `json:"tokenName"`
	TokenSymbol      string `json:"tokenSymbol"`
	TokenDecimal     string `json:"tokenDecimal"`
	TransactionIndex string `json:"transactionIndex"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	GasUsed          string `json:"gasUsed"`
}
