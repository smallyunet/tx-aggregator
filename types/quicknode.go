package types

// ---------------------------- JSON-RPC payload/response ------------------

// quickNodeTxRequest models qn_getTransactionsByAddress
type quickNodeTxRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// quickNodeTxResponse models the response structure
type quickNodeTxResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Address      string                 `json:"address"`
		EnsName      string                 `json:"ensName"`
		Transactions []quickNodeTransaction `json:"transactions"`
	} `json:"result"`
}

type quickNodeTransaction struct {
	BlockTimestamp   string `json:"blockTimestamp"`
	TransactionHash  string `json:"transactionHash"`
	BlockNumber      string `json:"blockNumber"`
	TransactionIndex string `json:"transactionIndex"`
	FromAddress      string `json:"fromAddress"`
	ToAddress        string `json:"toAddress"`
	ContractAddress  string `json:"contractAddress"`
	Value            string `json:"value"`
	Status           string `json:"status"`
}

// -------------------------- JSON-RPC models ------------------------------

type quickNodeTokenReq struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type quickNodeTokenResp struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Address string `json:"address"`
		EnsName string `json:"ensName"`

		Token struct {
			Address         string `json:"address"`
			Name            string `json:"name"`
			Symbol          string `json:"symbol"`
			Decimals        string `json:"decimals"`
			ContractAddress string `json:"contractAddress"`
		} `json:"token"`

		Transfers  []quickNodeTransfer `json:"transfers"`
		PageNumber int                 `json:"pageNumber"`
	} `json:"result"`
}

type quickNodeTransfer struct {
	Timestamp                    string `json:"timestamp"`
	BlockNumber                  string `json:"blockNumber"`
	TransactionHash              string `json:"transactionHash"`
	FromAddress                  string `json:"fromAddress"`
	ToAddress                    string `json:"toAddress"`
	SentAmount                   string `json:"sentAmount"`
	ReceivedAmount               string `json:"receivedAmount"`
	DecimalSentAmount            string `json:"decimalSentAmount"`
	DecimalReceivedAmount        string `json:"decimalReceivedAmount"`
	SentTokenContractAddress     string `json:"sentTokenContractAddress"`
	ReceivedTokenContractAddress string `json:"receivedTokenContractAddress"`
	Type                         string `json:"type"`
}
