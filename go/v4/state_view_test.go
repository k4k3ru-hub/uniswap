package v4

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/k4k3ru-hub/uniswap/go/v4/protocol"
)

var errTestCallContract = errors.New("test call contract error")

type stateViewHTTPRPCClient struct {
	call        ethereum.CallMsg
	blockNumber *big.Int
	result      []byte
	err         error
}

func (*stateViewHTTPRPCClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}

func (c *stateViewHTTPRPCClient) CallContract(_ context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	c.call = call
	c.blockNumber = blockNumber
	return c.result, c.err
}

func TestGetSlot0(t *testing.T) {
	t.Parallel()

	poolKey := testPoolKey(2)
	poolID, err := poolKey.ID()
	if err != nil {
		t.Fatalf("PoolKey.ID() error = %v", err)
	}

	want := Slot0{
		SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96),
		Tick:         -139,
		ProtocolFee:  500,
		LPFee:        3_000,
	}
	rpc := &stateViewHTTPRPCClient{result: encodeSlot0Result(t, want)}
	params := validClientParams()
	params.HTTPRPCClient = rpc
	params.PoolManager.PoolKeys = []protocol.PoolKey{poolKey}
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	blockNumber := big.NewInt(12_345)

	got, err := client.HTTP.GetSlot0(context.Background(), poolID, blockNumber)
	if err != nil {
		t.Fatalf("GetSlot0() error = %v", err)
	}
	if got.SqrtPriceX96.Cmp(want.SqrtPriceX96) != 0 || got.Tick != want.Tick || got.ProtocolFee != want.ProtocolFee || got.LPFee != want.LPFee {
		t.Fatalf("GetSlot0() = %+v, want %+v", got, want)
	}
	if rpc.call.To == nil || *rpc.call.To != params.PoolManager.StateView {
		t.Fatalf("call.To = %v, want %s", rpc.call.To, params.PoolManager.StateView.Hex())
	}
	if rpc.blockNumber != blockNumber {
		t.Fatalf("blockNumber = %v, want %v", rpc.blockNumber, blockNumber)
	}

	wantSelector := crypto.Keccak256([]byte(getSlot0Signature))[:4]
	if len(rpc.call.Data) != 36 {
		t.Fatalf("calldata length = %d, want 36", len(rpc.call.Data))
	}
	if string(rpc.call.Data[:4]) != string(wantSelector) {
		t.Fatalf("selector = %x, want %x", rpc.call.Data[:4], wantSelector)
	}
	if common.BytesToHash(rpc.call.Data[4:]) != poolID.Hash() {
		t.Fatalf("encoded pool ID = %s, want %s", common.BytesToHash(rpc.call.Data[4:]).Hex(), poolID.Hex())
	}
}

func TestGetSlot0AllowsLatestBlock(t *testing.T) {
	t.Parallel()

	poolKey := testPoolKey(2)
	poolID, err := poolKey.ID()
	if err != nil {
		t.Fatalf("PoolKey.ID() error = %v", err)
	}
	rpc := &stateViewHTTPRPCClient{result: encodeSlot0Result(t, Slot0{SqrtPriceX96: big.NewInt(1)})}
	params := validClientParams()
	params.HTTPRPCClient = rpc
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := client.HTTP.GetSlot0(context.Background(), poolID, nil); err != nil {
		t.Fatalf("GetSlot0() error = %v", err)
	}
	if rpc.blockNumber != nil {
		t.Fatalf("blockNumber = %v, want nil", rpc.blockNumber)
	}
}

func TestGetSlot0RejectsInvalidState(t *testing.T) {
	t.Parallel()

	poolKey := testPoolKey(2)
	poolID, err := poolKey.ID()
	if err != nil {
		t.Fatalf("PoolKey.ID() error = %v", err)
	}

	tests := []struct {
		name      string
		client    *HTTPClient
		poolID    protocol.PoolID
		wantError string
	}{
		{name: "nil client", client: nil, poolID: poolID, wantError: "client=null"},
		{name: "empty pool ID", client: newStateViewClient(t, nil, nil), poolID: protocol.PoolID{}, wantError: "pool_id=empty"},
		{name: "unconfigured pool ID", client: newStateViewClient(t, nil, nil), poolID: protocol.NewPoolID(common.HexToHash("0x01")), wantError: "pool_id=unconfigured"},
		{name: "malformed result", client: newStateViewClient(t, []byte{1}, nil), poolID: poolID, wantError: "failed to decode state view getSlot0 result"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := test.client.GetSlot0(context.Background(), test.poolID, nil)
			if err == nil {
				t.Fatal("GetSlot0() error = nil")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("GetSlot0() error = %q, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestGetSlot0WrapsCallError(t *testing.T) {
	t.Parallel()

	poolKey := testPoolKey(2)
	poolID, err := poolKey.ID()
	if err != nil {
		t.Fatalf("PoolKey.ID() error = %v", err)
	}
	client := newStateViewClient(t, nil, errTestCallContract)

	_, err = client.GetSlot0(context.Background(), poolID, nil)
	if err == nil {
		t.Fatal("GetSlot0() error = nil")
	}
	if !errors.Is(err, errTestCallContract) {
		t.Fatalf("errors.Is() = false, want wrapped call error: %v", err)
	}
}

func newStateViewClient(t *testing.T, result []byte, callErr error) *HTTPClient {
	t.Helper()

	params := validClientParams()
	client, err := NewHTTPClient(HTTPClientParams{
		RPC:         &stateViewHTTPRPCClient{result: result, err: callErr},
		PoolManager: params.PoolManager,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func encodeSlot0Result(t *testing.T, slot0 Slot0) []byte {
	t.Helper()

	uint160Type, err := abi.NewType("uint160", "", nil)
	if err != nil {
		t.Fatalf("abi.NewType(uint160) error = %v", err)
	}
	int24Type, err := abi.NewType("int24", "", nil)
	if err != nil {
		t.Fatalf("abi.NewType(int24) error = %v", err)
	}
	uint24Type, err := abi.NewType("uint24", "", nil)
	if err != nil {
		t.Fatalf("abi.NewType(uint24) error = %v", err)
	}

	result, err := (abi.Arguments{
		{Type: uint160Type},
		{Type: int24Type},
		{Type: uint24Type},
		{Type: uint24Type},
	}).Pack(
		slot0.SqrtPriceX96,
		big.NewInt(int64(slot0.Tick)),
		new(big.Int).SetUint64(uint64(slot0.ProtocolFee)),
		new(big.Int).SetUint64(uint64(slot0.LPFee)),
	)
	if err != nil {
		t.Fatalf("Arguments.Pack() error = %v", err)
	}
	return result
}
