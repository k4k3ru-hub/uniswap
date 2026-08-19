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
	Quoter    common.Address
	PoolKeys  []protocol.PoolKey
}

type HTTPClientParams struct {
	RPC         HTTPRPCClient
	PoolManager PoolManagerConfig
}

type WSClientParams struct {
	RPC         WSRPCClient
	PoolManager PoolManagerConfig
}

type ClientParams struct {
	HTTPRPCClient HTTPRPCClient
	WSRPCClient   WSRPCClient
	PoolManager   PoolManagerConfig
}

type HTTPClient struct {
	rpc         HTTPRPCClient
	poolManager common.Address
	stateView   common.Address
	quoter      common.Address
	poolKeys    []protocol.PoolKey
}

type WSClient struct {
	rpc         WSRPCClient
	poolManager common.Address
	poolKeys    []protocol.PoolKey
}

type Client struct {
	HTTP *HTTPClient
	WS   *WSClient
}

// NewHTTPClient creates a Uniswap v4 HTTP client.
//
// Parameters:
//   - params: HTTP RPC dependency and PoolManager configuration.
//
// Returns:
//   - Uniswap v4 HTTP client.
//   - Client creation error.
//
// Version:
//   - 2026-08-19: Required a V4Quoter deployment for HTTP quote operations.
//   - 2026-08-18: Added.
func NewHTTPClient(params HTTPClientParams) (*HTTPClient, error) {
	if params.RPC == nil {
		return nil, fmt.Errorf("failed to create uniswap v4 http client: http_rpc_client=null")
	}
	if params.PoolManager.Address == (common.Address{}) {
		return nil, fmt.Errorf("failed to create uniswap v4 http client: pool_manager=empty")
	}
	if params.PoolManager.StateView == (common.Address{}) {
		return nil, fmt.Errorf("failed to create uniswap v4 http client: state_view=empty")
	}
	if params.PoolManager.Quoter == (common.Address{}) {
		return nil, fmt.Errorf("failed to create uniswap v4 http client: quoter=empty")
	}

	poolKeys, err := validateAndCopyPoolKeys(params.PoolManager.PoolKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to create uniswap v4 http client: %w", err)
	}

	return &HTTPClient{
		rpc:         params.RPC,
		poolManager: params.PoolManager.Address,
		stateView:   params.PoolManager.StateView,
		quoter:      params.PoolManager.Quoter,
		poolKeys:    poolKeys,
	}, nil
}

// NewWSClient creates a Uniswap v4 WebSocket client.
//
// Parameters:
//   - params: WebSocket RPC dependency and PoolManager configuration.
//
// Returns:
//   - Uniswap v4 WebSocket client.
//   - Client creation error.
//
// Version:
//   - 2026-08-18: Added.
func NewWSClient(params WSClientParams) (*WSClient, error) {
	if params.RPC == nil {
		return nil, fmt.Errorf("failed to create uniswap v4 ws client: ws_rpc_client=null")
	}
	if params.PoolManager.Address == (common.Address{}) {
		return nil, fmt.Errorf("failed to create uniswap v4 ws client: pool_manager=empty")
	}

	poolKeys, err := validateAndCopyPoolKeys(params.PoolManager.PoolKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to create uniswap v4 ws client: %w", err)
	}

	return &WSClient{
		rpc:         params.RPC,
		poolManager: params.PoolManager.Address,
		poolKeys:    poolKeys,
	}, nil
}

// NewClient creates a complete Uniswap v4 client.
//
// Parameters:
//   - params: HTTP and WebSocket dependencies and PoolManager configuration.
//
// Returns:
//   - Uniswap v4 client composed from required HTTP and WebSocket clients.
//   - Client creation error.
//
// Version:
//   - 2026-08-19: Composed the required V4Quoter deployment through the HTTP client.
//   - 2026-08-18: Composed dedicated HTTP and WebSocket clients.
//   - 2026-08-17: Added.
func NewClient(params ClientParams) (*Client, error) {
	httpClient, err := NewHTTPClient(HTTPClientParams{
		RPC:         params.HTTPRPCClient,
		PoolManager: params.PoolManager,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create uniswap v4 client: %w", err)
	}

	wsClient, err := NewWSClient(WSClientParams{
		RPC:         params.WSRPCClient,
		PoolManager: params.PoolManager,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create uniswap v4 client: %w", err)
	}

	return &Client{
		HTTP: httpClient,
		WS:   wsClient,
	}, nil
}

func validateAndCopyPoolKeys(poolKeys []protocol.PoolKey) ([]protocol.PoolKey, error) {
	if len(poolKeys) == 0 {
		return nil, fmt.Errorf("failed to validate uniswap v4 pool keys: pool_keys=empty")
	}

	copiedPoolKeys := make([]protocol.PoolKey, len(poolKeys))
	poolIDs := make(map[protocol.PoolID]struct{}, len(poolKeys))
	for i, poolKey := range poolKeys {
		if err := poolKey.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate uniswap v4 pool keys: %w: pool_index=%d", err, i)
		}

		poolID, err := poolKey.ID()
		if err != nil {
			return nil, fmt.Errorf("failed to validate uniswap v4 pool keys: %w: pool_index=%d", err, i)
		}
		if _, exists := poolIDs[poolID]; exists {
			return nil, fmt.Errorf(
				"failed to validate uniswap v4 pool keys: duplicate pool key: pool_id=%q pool_index=%d",
				poolID.Hex(),
				i,
			)
		}

		poolIDs[poolID] = struct{}{}
		copiedPoolKeys[i] = poolKey
	}

	return copiedPoolKeys, nil
}

func hasPoolID(poolKeys []protocol.PoolKey, poolID protocol.PoolID) (bool, error) {
	for i, poolKey := range poolKeys {
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

func poolIDTopics(poolKeys []protocol.PoolKey) ([]common.Hash, error) {
	topics := make([]common.Hash, 0, len(poolKeys))
	for i, poolKey := range poolKeys {
		poolID, err := poolKey.ID()
		if err != nil {
			return nil, fmt.Errorf("failed to build uniswap v4 pool id topics: %w: pool_index=%d", err, i)
		}
		topics = append(topics, poolID.Hash())
	}

	return topics, nil
}
