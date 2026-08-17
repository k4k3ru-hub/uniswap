package v4

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/k4k3ru-hub/uniswap/go/v4/protocol"
)

const getSlot0Signature = "getSlot0(bytes32)"

type Slot0 struct {
	SqrtPriceX96 *big.Int
	Tick         int32
	ProtocolFee  uint32
	LPFee        uint32
}

// GetSlot0 gets the current Uniswap v4 pool state from StateView.
//
// Parameters:
//   - ctx: request context; nil uses context.Background in the injected HTTP RPC client.
//   - poolID: configured Uniswap v4 pool ID.
//   - blockNumber: block number; nil uses the latest block.
//
// Returns:
//   - Pool Slot0 state.
//   - State retrieval error.
//
// Version:
//   - 2026-08-17: Added.
func (c *Client) GetSlot0(ctx context.Context, poolID protocol.PoolID, blockNumber *big.Int) (Slot0, error) {
	if c == nil {
		return Slot0{}, fmt.Errorf("failed to get uniswap v4 slot0: client=null")
	}
	if c.httpRPCClient == nil {
		return Slot0{}, fmt.Errorf("failed to get uniswap v4 slot0: http_rpc_client=null")
	}
	if poolID.IsZero() {
		return Slot0{}, fmt.Errorf("failed to get uniswap v4 slot0: pool_id=empty")
	}

	configured, err := c.hasPoolID(poolID)
	if err != nil {
		return Slot0{}, fmt.Errorf("failed to get uniswap v4 slot0: %w", err)
	}
	if !configured {
		return Slot0{}, fmt.Errorf("failed to get uniswap v4 slot0: pool_id=unconfigured")
	}

	data, err := encodeGetSlot0Call(poolID)
	if err != nil {
		return Slot0{}, fmt.Errorf("failed to get uniswap v4 slot0: %w", err)
	}

	stateView := c.stateView
	result, err := c.httpRPCClient.CallContract(ctx, ethereum.CallMsg{
		To:   &stateView,
		Data: data,
	}, blockNumber)
	if err != nil {
		return Slot0{}, fmt.Errorf("failed to get uniswap v4 slot0: %w", err)
	}

	slot0, err := decodeGetSlot0Result(result)
	if err != nil {
		return Slot0{}, fmt.Errorf("failed to get uniswap v4 slot0: %w", err)
	}

	return slot0, nil
}

func (c *Client) hasPoolID(poolID protocol.PoolID) (bool, error) {
	for i, poolKey := range c.poolKeys {
		configuredPoolID, err := poolKey.ID()
		if err != nil {
			return false, fmt.Errorf("failed to match uniswap v4 pool id: %w: pool_index=%d", err, i)
		}
		if configuredPoolID == poolID {
			return true, nil
		}
	}

	return false, nil
}

func encodeGetSlot0Call(poolID protocol.PoolID) ([]byte, error) {
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encode state view getSlot0 call: %w: abi_type=%q", err, "bytes32")
	}

	encodedPoolID, err := (abi.Arguments{{Type: bytes32Type}}).Pack(poolID.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to encode state view getSlot0 call: %w", err)
	}

	selector := crypto.Keccak256([]byte(getSlot0Signature))[:4]
	data := make([]byte, 0, len(selector)+len(encodedPoolID))
	data = append(data, selector...)
	data = append(data, encodedPoolID...)

	return data, nil
}

func decodeGetSlot0Result(result []byte) (Slot0, error) {
	uint160Type, err := abi.NewType("uint160", "", nil)
	if err != nil {
		return Slot0{}, fmt.Errorf("failed to decode state view getSlot0 result: %w: abi_type=%q", err, "uint160")
	}
	int24Type, err := abi.NewType("int24", "", nil)
	if err != nil {
		return Slot0{}, fmt.Errorf("failed to decode state view getSlot0 result: %w: abi_type=%q", err, "int24")
	}
	uint24Type, err := abi.NewType("uint24", "", nil)
	if err != nil {
		return Slot0{}, fmt.Errorf("failed to decode state view getSlot0 result: %w: abi_type=%q", err, "uint24")
	}

	values, err := (abi.Arguments{
		{Type: uint160Type},
		{Type: int24Type},
		{Type: uint24Type},
		{Type: uint24Type},
	}).Unpack(result)
	if err != nil {
		return Slot0{}, fmt.Errorf("failed to decode state view getSlot0 result: %w", err)
	}
	if len(values) != 4 {
		return Slot0{}, fmt.Errorf("failed to decode state view getSlot0 result: values=invalid actual_length=%d expected_length=4", len(values))
	}

	sqrtPriceX96, ok := values[0].(*big.Int)
	if !ok || sqrtPriceX96.Sign() < 0 || sqrtPriceX96.BitLen() > 160 {
		return Slot0{}, fmt.Errorf("failed to decode state view getSlot0 result: sqrt_price_x96=invalid")
	}
	tick, ok := values[1].(*big.Int)
	if !ok || !tick.IsInt64() || tick.Int64() < -(1<<23) || tick.Int64() > (1<<23)-1 {
		return Slot0{}, fmt.Errorf("failed to decode state view getSlot0 result: tick=invalid")
	}
	protocolFee, ok := values[2].(*big.Int)
	if !ok || protocolFee.Sign() < 0 || protocolFee.BitLen() > 24 {
		return Slot0{}, fmt.Errorf("failed to decode state view getSlot0 result: protocol_fee=invalid")
	}
	lpFee, ok := values[3].(*big.Int)
	if !ok || lpFee.Sign() < 0 || lpFee.BitLen() > 24 {
		return Slot0{}, fmt.Errorf("failed to decode state view getSlot0 result: lp_fee=invalid")
	}

	return Slot0{
		SqrtPriceX96: new(big.Int).Set(sqrtPriceX96),
		Tick:         int32(tick.Int64()),
		ProtocolFee:  uint32(protocolFee.Uint64()),
		LPFee:        uint32(lpFee.Uint64()),
	}, nil
}
