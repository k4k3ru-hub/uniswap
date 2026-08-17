package v4

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/k4k3ru-hub/uniswap/go/v4/protocol"
)

var errTestFilterLogs = errors.New("test filter logs error")

type filterSwapHTTPRPCClient struct {
	query ethereum.FilterQuery
	logs  []types.Log
	err   error
}

func (c *filterSwapHTTPRPCClient) FilterLogs(_ context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	c.query = query
	return c.logs, c.err
}

func (*filterSwapHTTPRPCClient) CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
	return nil, nil
}

func TestFilterSwaps(t *testing.T) {
	t.Parallel()

	params := validClientParams()
	poolKey0 := testPoolKey(2)
	poolKey1 := testPoolKey(4)
	params.PoolManager.PoolKeys = []protocol.PoolKey{poolKey0, poolKey1}
	poolID0, err := poolKey0.ID()
	if err != nil {
		t.Fatalf("poolKey0.ID() error = %v", err)
	}
	poolID1, err := poolKey1.ID()
	if err != nil {
		t.Fatalf("poolKey1.ID() error = %v", err)
	}

	eventLog0 := newSwapLog(t)
	eventLog0.Address = params.PoolManager.Address
	eventLog0.Topics[1] = poolID0.Hash()
	eventLog1 := cloneLog(eventLog0)
	eventLog1.Topics[1] = poolID1.Hash()
	eventLog1.Index = 1
	rpc := &filterSwapHTTPRPCClient{logs: []types.Log{eventLog0, eventLog1}}
	params.HTTPRPCClient = rpc
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	fromBlock := big.NewInt(100)
	toBlock := big.NewInt(200)

	swaps, err := client.HTTP.FilterSwaps(context.Background(), fromBlock, toBlock)
	if err != nil {
		t.Fatalf("FilterSwaps() error = %v", err)
	}
	if len(swaps) != 2 || swaps[0].PoolID != poolID0 || swaps[1].PoolID != poolID1 {
		t.Fatalf("FilterSwaps() pools = %+v, want [%s, %s]", swaps, poolID0.Hex(), poolID1.Hex())
	}
	if rpc.query.FromBlock != fromBlock || rpc.query.ToBlock != toBlock {
		t.Fatalf("query blocks = (%v, %v), want (%v, %v)", rpc.query.FromBlock, rpc.query.ToBlock, fromBlock, toBlock)
	}
	if len(rpc.query.Addresses) != 1 || rpc.query.Addresses[0] != params.PoolManager.Address {
		t.Fatalf("query addresses = %v, want [%s]", rpc.query.Addresses, params.PoolManager.Address.Hex())
	}
	if len(rpc.query.Topics) != 2 || len(rpc.query.Topics[0]) != 1 || rpc.query.Topics[0][0] != swapEventSignatureHash() {
		t.Fatalf("query event topics = %v", rpc.query.Topics)
	}
	if len(rpc.query.Topics[1]) != 2 || rpc.query.Topics[1][0] != poolID0.Hash() || rpc.query.Topics[1][1] != poolID1.Hash() {
		t.Fatalf("query pool topics = %v", rpc.query.Topics[1])
	}
}

func TestFilterSwapsAllowsDefaultBlockRange(t *testing.T) {
	t.Parallel()

	params := validClientParams()
	rpc := &filterSwapHTTPRPCClient{}
	params.HTTPRPCClient = rpc
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	swaps, err := client.HTTP.FilterSwaps(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("FilterSwaps() error = %v", err)
	}
	if len(swaps) != 0 {
		t.Fatalf("FilterSwaps() length = %d, want 0", len(swaps))
	}
	if rpc.query.FromBlock != nil || rpc.query.ToBlock != nil {
		t.Fatalf("query blocks = (%v, %v), want nil", rpc.query.FromBlock, rpc.query.ToBlock)
	}
}

func TestFilterSwapsRejectsUnexpectedLogs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*types.Log)
		wantError string
	}{
		{name: "wrong pool manager", mutate: func(log *types.Log) { log.Address = common.HexToAddress("0x01") }, wantError: "pool_manager=invalid"},
		{name: "unconfigured pool", mutate: func(log *types.Log) { log.Topics[1] = common.HexToHash("0x123456") }, wantError: "pool_id=unconfigured"},
		{name: "invalid swap log", mutate: func(log *types.Log) { log.Topics[0] = common.Hash{} }, wantError: "event_signature=invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			params := validClientParams()
			poolID, err := params.PoolManager.PoolKeys[0].ID()
			if err != nil {
				t.Fatalf("PoolKey.ID() error = %v", err)
			}
			eventLog := newSwapLog(t)
			eventLog.Address = params.PoolManager.Address
			eventLog.Topics[1] = poolID.Hash()
			test.mutate(&eventLog)
			params.HTTPRPCClient = &filterSwapHTTPRPCClient{logs: []types.Log{eventLog}}
			client, err := NewClient(params)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			_, err = client.HTTP.FilterSwaps(context.Background(), nil, nil)
			if err == nil {
				t.Fatal("FilterSwaps() error = nil")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("FilterSwaps() error = %q, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestFilterSwapsWrapsRPCError(t *testing.T) {
	t.Parallel()

	params := validClientParams()
	params.HTTPRPCClient = &filterSwapHTTPRPCClient{err: errTestFilterLogs}
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.HTTP.FilterSwaps(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("FilterSwaps() error = nil")
	}
	if !errors.Is(err, errTestFilterLogs) {
		t.Fatalf("errors.Is() = false, want wrapped filter error: %v", err)
	}
}
