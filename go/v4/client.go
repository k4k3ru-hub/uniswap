package v4

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/k4k3ru-hub/uniswap/go/v4/protocol"
)

type HTTPRPCClient interface {
	FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error)
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

type WSRPCClient interface {
	SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
}

type PoolManagerConfig struct {
	Address   common.Address
	StateView common.Address
	PoolKeys  []protocol.PoolKey
}

type ClientParams struct {
	HTTPRPCClient HTTPRPCClient
	WSRPCClient   WSRPCClient
	PoolManager   PoolManagerConfig
}

type Client struct {
	httpRPCClient HTTPRPCClient
	wsRPCClient   WSRPCClient
	poolManager   common.Address
	stateView     common.Address
	poolKeys      []protocol.PoolKey
}

// NewClient creates a Uniswap v4 client.
//
// Parameters:
//   - params: HTTP and WebSocket dependencies and PoolManager configuration.
//
// Returns:
//   - Uniswap v4 client.
//   - Client creation error.
//
// Version:
//   - 2026-08-17: Added.
func NewClient(params ClientParams) (*Client, error) {
	if params.HTTPRPCClient == nil {
		return nil, fmt.Errorf("failed to create uniswap v4 client: http_rpc_client=null")
	}
	if params.WSRPCClient == nil {
		return nil, fmt.Errorf("failed to create uniswap v4 client: ws_rpc_client=null")
	}
	if params.PoolManager.Address == (common.Address{}) {
		return nil, fmt.Errorf("failed to create uniswap v4 client: pool_manager=empty")
	}
	if params.PoolManager.StateView == (common.Address{}) {
		return nil, fmt.Errorf("failed to create uniswap v4 client: state_view=empty")
	}
	if len(params.PoolManager.PoolKeys) == 0 {
		return nil, fmt.Errorf("failed to create uniswap v4 client: pool_keys=empty")
	}

	poolKeys := make([]protocol.PoolKey, len(params.PoolManager.PoolKeys))
	poolIDs := make(map[protocol.PoolID]struct{}, len(params.PoolManager.PoolKeys))
	for i, poolKey := range params.PoolManager.PoolKeys {
		if err := poolKey.Validate(); err != nil {
			return nil, fmt.Errorf("failed to create uniswap v4 client: %w: pool_index=%d", err, i)
		}

		poolID, err := poolKey.ID()
		if err != nil {
			return nil, fmt.Errorf("failed to create uniswap v4 client: %w: pool_index=%d", err, i)
		}
		if _, exists := poolIDs[poolID]; exists {
			return nil, fmt.Errorf(
				"failed to create uniswap v4 client: duplicate pool key: pool_id=%q pool_index=%d",
				poolID.Hex(),
				i,
			)
		}

		poolIDs[poolID] = struct{}{}
		poolKeys[i] = poolKey
	}

	return &Client{
		httpRPCClient: params.HTTPRPCClient,
		wsRPCClient:   params.WSRPCClient,
		poolManager:   params.PoolManager.Address,
		stateView:     params.PoolManager.StateView,
		poolKeys:      poolKeys,
	}, nil
}
