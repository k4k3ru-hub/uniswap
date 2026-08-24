# Uniswap Go clients

Read-only Go clients for Uniswap v3 and v4. Both versions provide independently composable HTTP and WebSocket clients, pool state reads, single-pool quotes, historical Swap filtering, and live Swap subscriptions.

The v3 deployment helper supports Ethereum (`1`), BNB Chain (`56`), and Base (`8453`). V3 pools accept ERC-20 token addresses only; use wrapped native tokens such as WETH rather than the zero address.

```sh
cd go
go run ./cli v3 get-slot0 \
  --http-url https://rpc.example \
  --chain-id 1 \
  --token0 0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48 \
  --token1 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2 \
  --fee 3000
```

Use `v4 get-slot0` with the existing v4 pool-key options for Uniswap v4 pools.
