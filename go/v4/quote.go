package v4

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/k4k3ru-hub/uniswap/go/v4/protocol"
)

const (
	quoteExactInputSingleSignature  = "quoteExactInputSingle(((address,address,uint24,int24,address),bool,uint128,bytes))"
	quoteExactOutputSingleSignature = "quoteExactOutputSingle(((address,address,uint24,int24,address),bool,uint128,bytes))"
)

type QuoteExactInputSingleParams struct {
	PoolKey protocol.PoolKey
	// ZeroForOne indicates currency0 to currency1 when true.
	ZeroForOne bool
	// AmountIn is the exact input amount in base units.
	AmountIn *big.Int
	HookData []byte
}

type QuoteExactInputSingleResult struct {
	// AmountOut is the quoted output amount in base units.
	AmountOut   *big.Int
	GasEstimate *big.Int
}

type QuoteExactOutputSingleParams struct {
	PoolKey protocol.PoolKey
	// ZeroForOne indicates currency0 to currency1 when true.
	ZeroForOne bool
	// AmountOut is the exact output amount in base units.
	AmountOut *big.Int
	HookData  []byte
}

type QuoteExactOutputSingleResult struct {
	// AmountIn is the quoted input amount in base units.
	AmountIn    *big.Int
	GasEstimate *big.Int
}

type quotePoolKeyABI struct {
	Currency0   common.Address
	Currency1   common.Address
	Fee         *big.Int
	TickSpacing *big.Int
	Hooks       common.Address
}

type quoteExactSingleParamsABI struct {
	PoolKey     quotePoolKeyABI
	ZeroForOne  bool
	ExactAmount *big.Int
	HookData    []byte
}

// QuoteExactInputSingle quotes a single-pool exact-input swap.
//
// The quote uses an unspecified EVM caller because CallMsg.From is not set. Hooks whose behavior depends on msg.sender may require a caller-aware API.
//
// Parameters:
//   - ctx: request context; nil is passed to the injected HTTP RPC client.
//   - params: pool and exact-input quote parameters; AmountIn is in base units.
//   - blockNumber: block number; nil uses the latest block.
//
// Returns:
//   - Exact-input quote result in base units.
//   - Quote error.
//
// Version:
//   - 2026-08-19: Added.
func (c *HTTPClient) QuoteExactInputSingle(ctx context.Context, params QuoteExactInputSingleParams, blockNumber *big.Int) (QuoteExactInputSingleResult, error) {
	const operation = "failed to quote uniswap v4 exact input single"
	if err := c.validateSingleQuote(params.PoolKey, params.AmountIn, "amount_in", blockNumber); err != nil {
		return QuoteExactInputSingleResult{}, fmt.Errorf("%s: %w", operation, err)
	}

	data, err := encodeQuoteExactSingleCall(quoteExactInputSingleSignature, params.PoolKey, params.ZeroForOne, params.AmountIn, params.HookData)
	if err != nil {
		return QuoteExactInputSingleResult{}, fmt.Errorf("%s: %w", operation, err)
	}
	quoter := c.quoter
	response, err := c.rpc.CallContract(ctx, ethereum.CallMsg{To: &quoter, Data: data}, blockNumber)
	if err != nil {
		return QuoteExactInputSingleResult{}, fmt.Errorf("%s: %w", operation, err)
	}
	amountOut, gasEstimate, err := decodeQuoteExactSingleResult(response, "amount_out", "failed to decode uniswap v4 exact input single result")
	if err != nil {
		return QuoteExactInputSingleResult{}, fmt.Errorf("%s: %w", operation, err)
	}
	return QuoteExactInputSingleResult{AmountOut: amountOut, GasEstimate: gasEstimate}, nil
}

// QuoteExactOutputSingle quotes a single-pool exact-output swap.
//
// The quote uses an unspecified EVM caller because CallMsg.From is not set. Hooks whose behavior depends on msg.sender may require a caller-aware API.
//
// Parameters:
//   - ctx: request context; nil is passed to the injected HTTP RPC client.
//   - params: pool and exact-output quote parameters; AmountOut is in base units.
//   - blockNumber: block number; nil uses the latest block.
//
// Returns:
//   - Exact-output quote result in base units.
//   - Quote error.
//
// Version:
//   - 2026-08-19: Added.
func (c *HTTPClient) QuoteExactOutputSingle(ctx context.Context, params QuoteExactOutputSingleParams, blockNumber *big.Int) (QuoteExactOutputSingleResult, error) {
	const operation = "failed to quote uniswap v4 exact output single"
	if err := c.validateSingleQuote(params.PoolKey, params.AmountOut, "amount_out", blockNumber); err != nil {
		return QuoteExactOutputSingleResult{}, fmt.Errorf("%s: %w", operation, err)
	}

	data, err := encodeQuoteExactSingleCall(quoteExactOutputSingleSignature, params.PoolKey, params.ZeroForOne, params.AmountOut, params.HookData)
	if err != nil {
		return QuoteExactOutputSingleResult{}, fmt.Errorf("%s: %w", operation, err)
	}
	quoter := c.quoter
	response, err := c.rpc.CallContract(ctx, ethereum.CallMsg{To: &quoter, Data: data}, blockNumber)
	if err != nil {
		return QuoteExactOutputSingleResult{}, fmt.Errorf("%s: %w", operation, err)
	}
	amountIn, gasEstimate, err := decodeQuoteExactSingleResult(response, "amount_in", "failed to decode uniswap v4 exact output single result")
	if err != nil {
		return QuoteExactOutputSingleResult{}, fmt.Errorf("%s: %w", operation, err)
	}
	return QuoteExactOutputSingleResult{AmountIn: amountIn, GasEstimate: gasEstimate}, nil
}

func (c *HTTPClient) validateSingleQuote(poolKey protocol.PoolKey, exactAmount *big.Int, amountName string, blockNumber *big.Int) error {
	if c == nil {
		return fmt.Errorf("failed to validate uniswap v4 single quote: client=null")
	}
	if c.rpc == nil {
		return fmt.Errorf("failed to validate uniswap v4 single quote: http_rpc_client=null")
	}
	if c.quoter == (common.Address{}) {
		return fmt.Errorf("failed to validate uniswap v4 single quote: quoter=empty")
	}
	if err := poolKey.Validate(); err != nil {
		return fmt.Errorf("failed to validate uniswap v4 single quote: %w", err)
	}
	poolID, err := poolKey.ID()
	if err != nil {
		return fmt.Errorf("failed to validate uniswap v4 single quote: %w", err)
	}
	configured, err := hasPoolID(c.poolKeys, poolID)
	if err != nil {
		return fmt.Errorf("failed to validate uniswap v4 single quote: %w", err)
	}
	if !configured {
		return fmt.Errorf("failed to validate uniswap v4 single quote: pool_id=unconfigured")
	}
	if exactAmount == nil {
		return fmt.Errorf("failed to validate uniswap v4 single quote: %s=null", amountName)
	}
	if exactAmount.Sign() <= 0 || exactAmount.BitLen() > 128 {
		return fmt.Errorf("failed to validate uniswap v4 single quote: %s=out_of_range", amountName)
	}
	if blockNumber != nil && blockNumber.Sign() < 0 {
		return fmt.Errorf("failed to validate uniswap v4 single quote: block_number=out_of_range")
	}
	return nil
}

func encodeQuoteExactSingleCall(signature string, poolKey protocol.PoolKey, zeroForOne bool, exactAmount *big.Int, hookData []byte) ([]byte, error) {
	operation := "failed to encode uniswap v4 exact input single call"
	if signature == quoteExactOutputSingleSignature {
		operation = "failed to encode uniswap v4 exact output single call"
	}
	paramsType, err := quoteExactSingleParamsType()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	params := quoteExactSingleParamsABI{
		PoolKey: quotePoolKeyABI{
			Currency0:   poolKey.Currency0.Address(),
			Currency1:   poolKey.Currency1.Address(),
			Fee:         new(big.Int).SetUint64(uint64(poolKey.Fee)),
			TickSpacing: big.NewInt(int64(poolKey.TickSpacing)),
			Hooks:       poolKey.Hooks,
		},
		ZeroForOne:  zeroForOne,
		ExactAmount: new(big.Int).Set(exactAmount),
		HookData:    append([]byte(nil), hookData...),
	}
	encoded, err := (abi.Arguments{{Type: paramsType}}).Pack(params)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	selector := crypto.Keccak256([]byte(signature))[:4]
	return append(append([]byte(nil), selector...), encoded...), nil
}

func quoteExactSingleParamsType() (abi.Type, error) {
	return abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "poolKey", Type: "tuple", Components: []abi.ArgumentMarshaling{
			{Name: "currency0", Type: "address"},
			{Name: "currency1", Type: "address"},
			{Name: "fee", Type: "uint24"},
			{Name: "tickSpacing", Type: "int24"},
			{Name: "hooks", Type: "address"},
		}},
		{Name: "zeroForOne", Type: "bool"},
		{Name: "exactAmount", Type: "uint128"},
		{Name: "hookData", Type: "bytes"},
	})
}

func decodeQuoteExactSingleResult(response []byte, amountName, operation string) (*big.Int, *big.Int, error) {
	if len(response) == 0 {
		return nil, nil, fmt.Errorf("%s: response=empty", operation)
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", operation, err)
	}
	values, err := (abi.Arguments{{Type: uint256Type}, {Type: uint256Type}}).Unpack(response)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", operation, err)
	}
	if len(values) != 2 {
		return nil, nil, fmt.Errorf("%s: values=invalid actual_length=%d expected_length=2", operation, len(values))
	}
	amount, ok := values[0].(*big.Int)
	if !ok || amount == nil || amount.Sign() <= 0 {
		return nil, nil, fmt.Errorf("%s: %s=invalid", operation, amountName)
	}
	gasEstimate, ok := values[1].(*big.Int)
	if !ok || gasEstimate == nil || gasEstimate.Sign() < 0 || gasEstimate.BitLen() > 256 {
		return nil, nil, fmt.Errorf("%s: gas_estimate=invalid", operation)
	}
	return new(big.Int).Set(amount), new(big.Int).Set(gasEstimate), nil
}
