# Uniswap Go clients

Read-only Go clients for Uniswap v3 and v4. Both versions provide independently composable HTTP and WebSocket clients, pool state reads, single-pool quotes, historical Swap filtering, and live Swap subscriptions.

The v3 deployment helper supports Ethereum (`1`), BNB Chain (`56`), Base (`8453`), and Robinhood Chain (`4663`). V3 pools accept ERC-20 token addresses only; use wrapped native tokens such as WETH rather than the zero address.

Robinhood Chain Mainnet supports both HTTP quote composition roots: v3 resolves
Factory and QuoterV2, and v4 resolves PoolManager, StateView, and V4Quoter. Pass an
RPC transport explicitly to the owning v3/v4 constructor. Testnet deployments are
not inferred from Mainnet addresses.

Deployment sources: [v3 contracts](https://docs.ponsfamily.com/#contracts) and
[official v4 deployments](https://developers.uniswap.org/docs/protocols/v4/deployments).
The Mainnet contracts and the PONS/USDG pools were checked through read-only RPC
calls on 2026-09-06. Pool existence alone is not a guarantee of current liquidity.

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
