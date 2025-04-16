# Tx Aggregator

Tx Aggregator is a blockchain transaction data aggregation service that collects, processes, and provides cross-chain transaction data. The service supports multiple blockchain networks and offers a unified API interface for querying transaction information.

## Key Features

- Multi-chain transaction data aggregation (ETH, BSC, etc.)
- Unified transaction query API
- Support for native tokens and ERC20/BEP20 token transactions
- Built-in Redis caching mechanism for improved query performance
- Transaction filtering and pagination
- Detailed transaction information including status, gas fees, etc.

## Tech Stack

- Go programming language
- Fiber web framework
- Redis caching
- Ankr API integration

## Quick Start

### Prerequisites

- Go 1.16+
- Redis
- Ankr API Key

### Installation

1. Clone the repository
```bash
git clone git@gitlab.devops.tantin.com:walletbackend/tantin_transaction_api.git
cd tantin_transaction_api
```

2. Run the service
```bash
make dev
```

## API Usage

### Get Transaction List

```
GET /transactions?address=<wallet_address>&chainName=<chain_name>&tokenAddress=<token_address>
```

Parameters:
- `address`: Wallet address (required)
- `chainName`: Chain name(s), comma-separated (optional, defaults to all supported chains)
- `tokenAddress`: Token contract address (optional, for filtering specific token transactions)

Example Response:
```json
{
  "code": 0,
  "message": "success",
  "result": {
    "transactions": [
      {
        "chainId": 1,
        "tokenId": 0,
        "state": 1,
        "height": 12345678,
        "hash": "0x...",
        "fromAddress": "0x...",
        "toAddress": "0x...",
        "amount": "1000000000000000000",
        "gasUsed": "21000",
        "gasPrice": "20000000000",
        "type": 0,
        "coinType": 1,
        "createdTime": 1234567890,
        "tranType": 0
      }
    ]
  }
}
```

## Project Structure

```
tx-aggregator/
├── api/            # API handlers
├── cache/          # Cache implementation
├── config/         # Configuration management
├── logger/         # Logging
├── model/          # Data models
├── provider/       # Data providers
├── router/         # Route definitions
├── types/          # Type definitions
└── usecase/        # Business logic
```

## Contributing

1. Fork the project
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the [MIT License](LICENSE).
