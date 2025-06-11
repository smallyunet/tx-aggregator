# Tx Aggregator

Tx Aggregator is a Go service that aggregates blockchain transactions from multiple data sources. It normalises
provider‑specific data into a single format, caches the result in Redis and exposes a simple HTTP API for queries.

The service is designed to run in different environments (local, dev, test, prod) and uses Consul for both service
registration and configuration. Logging is handled by [zerolog](https://github.com/rs/zerolog).

## Features

- **Multi‑chain support** – fetches transactions from Ethereum, BSC, Polygon and other chains defined in the config.
- **Pluggable providers** – built‑in connectors for Ankr, Blockscout, Blockscan and QuickNode. Each chain can be routed
to a different provider.
- **Redis cache** – transactions are grouped by address/chain/token and cached for faster repeated queries.
- **Consul integration** – runtime configuration and service registration are performed via Consul; settings are hot‑reloaded.
- **Unified API** – a single `/transactions` endpoint returns native and ERC‑20/BEP‑20 transfers with gas details.
- **Graceful shutdown** – the process deregisters from Consul and exits cleanly on SIGINT/SIGTERM.

## Directory overview

```
tx-aggregator/
├── api/            # HTTP handlers and parameter parsing
├── cache/          # Redis helpers and transaction caching logic
├── cmd/            # Main applications (service and integration tests)
├── config/         # Runtime configuration loader
├── configfiles/    # Example bootstrap/config YAML files
├── consul/         # Consul bootstrap loader and service registrar
├── provider/       # Provider implementations (ankr, blockscout, ...)
├── router/         # Route definitions for Fiber
├── types/          # Shared type definitions
└── usecase/        # Business logic (filtering, sorting, etc.)
```

## Configuration

Startup uses two YAML files:

1. **Bootstrap** (`bootstrap.<env>.yaml`) – Consul address, service name and port.
2. **Runtime config** (`config/<service>/<env>`) – loaded from Consul KV (with optional local override). It defines
   Redis settings, provider mappings and other behaviour. A sample is located under `configfiles/`.

Configuration is hot‑reloaded every 10 seconds. Environment is selected via the `APP_ENV` variable.

## Running

Install dependencies and start the service in development mode:

```bash
make dev               # hot‑reload with Air (APP_ENV=local)
```

To run once without hot‑reload:

```bash
make run APP_ENV=dev
```

The service listens on the port defined in the runtime configuration (default `8080`).

## API

### `GET /transactions`

Query parameters:

- `address` – wallet address (required)
- `chainName` – comma‑separated list of chain names (optional; defaults to all supported chains)
- `tokenAddress` – token contract address or `native` (optional)

Example:

```
GET /transactions?address=0x123...&chainName=ETH,BSC&tokenAddress=native
```

The response body always has HTTP status `200`. Application errors are reported via the `code` field.

```json
{
  "code": 0,
  "message": "success",
  "result": {
    "transactions": [
      {
        "chainId": 1,
        "hash": "0x...",
        "fromAddress": "0x...",
        "toAddress": "0x...",
        "amount": "1000000000000000000",
        "gasUsed": "21000",
        "gasPrice": "20000000000",
        "coinType": 1,
        "tranType": 0
      }
    ]
  }
}
```

Transactions are sorted by block height and limited to the `response.max` value from the configuration.

## Testing

Unit tests can be run with:

```bash
go test ./...
```

An integration test tool is available under `cmd/tx-aggregator-integration`. It compares API responses across
environments and checks transaction counts.

## License

This project is licensed under the [MIT License](LICENSE).
