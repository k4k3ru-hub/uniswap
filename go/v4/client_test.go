package v4

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	myOnchainEVM "github.com/k4k3ru-hub/onchain/go/evm"
	"github.com/k4k3ru-hub/uniswap/go/v4/protocol"
)

var (
	_ HTTPRPCClient = (*myOnchainEVM.HTTPClient)(nil)
	_ WSRPCClient   = (*myOnchainEVM.WSClient)(nil)
)

type fakeHTTPClient struct{}

func (*fakeHTTPClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}

func (*fakeHTTPClient) CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
	return nil, nil
}

type fakeWSClient struct{}

func (*fakeWSClient) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	return nil, nil
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	httpClient := &fakeHTTPClient{}
	wsClient := &fakeWSClient{}
	poolKey := testPoolKey(2)
	poolManager := common.HexToAddress("0x0000000000000000000000000000000000000003")
	stateView := common.HexToAddress("0x0000000000000000000000000000000000000004")

	client, err := NewClient(ClientParams{
		HTTPRPCClient: httpClient,
		WSRPCClient:   wsClient,
		PoolManager: PoolManagerConfig{
			Address:   poolManager,
			StateView: stateView,
			PoolKeys:  []protocol.PoolKey{poolKey},
		},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.httpRPCClient != httpClient {
		t.Fatal("httpRPCClient was not composed")
	}
	if client.wsRPCClient != wsClient {
		t.Fatal("wsRPCClient was not composed")
	}
	if client.poolManager != poolManager {
		t.Fatalf("poolManager = %s, want %s", client.poolManager.Hex(), poolManager.Hex())
	}
	if client.stateView != stateView {
		t.Fatalf("stateView = %s, want %s", client.stateView.Hex(), stateView.Hex())
	}
	if len(client.poolKeys) != 1 || client.poolKeys[0] != poolKey {
		t.Fatalf("poolKeys = %+v, want [%+v]", client.poolKeys, poolKey)
	}
}

func TestNewClientRejectsInvalidParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*ClientParams)
		wantError string
	}{
		{
			name: "missing HTTP RPC client",
			mutate: func(params *ClientParams) {
				params.HTTPRPCClient = nil
			},
			wantError: "failed to create uniswap v4 client: http_rpc_client=null",
		},
		{
			name: "missing WebSocket RPC client",
			mutate: func(params *ClientParams) {
				params.WSRPCClient = nil
			},
			wantError: "failed to create uniswap v4 client: ws_rpc_client=null",
		},
		{
			name: "empty pool manager",
			mutate: func(params *ClientParams) {
				params.PoolManager.Address = common.Address{}
			},
			wantError: "failed to create uniswap v4 client: pool_manager=empty",
		},
		{
			name: "empty state view",
			mutate: func(params *ClientParams) {
				params.PoolManager.StateView = common.Address{}
			},
			wantError: "failed to create uniswap v4 client: state_view=empty",
		},
		{
			name: "empty pool keys",
			mutate: func(params *ClientParams) {
				params.PoolManager.PoolKeys = nil
			},
			wantError: "failed to create uniswap v4 client: pool_keys=empty",
		},
		{
			name: "invalid pool key",
			mutate: func(params *ClientParams) {
				params.PoolManager.PoolKeys[0].TickSpacing = 0
			},
			wantError: "failed to create uniswap v4 client: failed to validate uniswap v4 pool key: tick_spacing=out_of_range min_value=1 max_value=32767: pool_index=0",
		},
		{
			name: "duplicate pool key",
			mutate: func(params *ClientParams) {
				params.PoolManager.PoolKeys = append(params.PoolManager.PoolKeys, params.PoolManager.PoolKeys[0])
			},
			wantError: "failed to create uniswap v4 client: duplicate pool key:",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			params := validClientParams()
			test.mutate(&params)
			_, err := NewClient(params)
			if err == nil {
				t.Fatal("NewClient() error = nil")
			}
			if !strings.HasPrefix(err.Error(), test.wantError) {
				t.Fatalf("NewClient() error = %q, want prefix %q", err, test.wantError)
			}
		})
	}
}

func TestNewClientCopiesPoolKeys(t *testing.T) {
	t.Parallel()

	params := validClientParams()
	originalPoolKey := params.PoolManager.PoolKeys[0]
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	params.PoolManager.PoolKeys[0] = testPoolKey(4)
	if client.poolKeys[0] != originalPoolKey {
		t.Fatalf("client pool key changed to %+v, want %+v", client.poolKeys[0], originalPoolKey)
	}
}

func validClientParams() ClientParams {
	return ClientParams{
		HTTPRPCClient: &fakeHTTPClient{},
		WSRPCClient:   &fakeWSClient{},
		PoolManager: PoolManagerConfig{
			Address:   common.HexToAddress("0x0000000000000000000000000000000000000003"),
			StateView: common.HexToAddress("0x0000000000000000000000000000000000000004"),
			PoolKeys:  []protocol.PoolKey{testPoolKey(2)},
		},
	}
}

func testPoolKey(currency1 byte) protocol.PoolKey {
	return protocol.PoolKey{
		Currency0:   protocol.NewCurrency(common.HexToAddress("0x0000000000000000000000000000000000000001")),
		Currency1:   protocol.NewCurrency(common.BytesToAddress([]byte{currency1})),
		Fee:         3_000,
		TickSpacing: 60,
	}
}
