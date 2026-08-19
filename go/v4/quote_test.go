package v4

import (
	"bytes"
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

type quoteHTTPRPCClient struct {
	response    []byte
	err         error
	call        ethereum.CallMsg
	blockNumber *big.Int
}

func (*quoteHTTPRPCClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}

func (c *quoteHTTPRPCClient) CallContract(_ context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	c.call = call
	c.blockNumber = blockNumber
	return c.response, c.err
}

func TestQuoteExactInputSingle(t *testing.T) {
	t.Parallel()
	testSingleQuote(t, true, true, nil)
	testSingleQuote(t, true, false, big.NewInt(123))
}

func TestQuoteExactOutputSingle(t *testing.T) {
	t.Parallel()
	testSingleQuote(t, false, true, nil)
	testSingleQuote(t, false, false, big.NewInt(456))
}

func testSingleQuote(t *testing.T, exactInput, zeroForOne bool, blockNumber *big.Int) {
	t.Helper()
	response := packQuoteResult(t, big.NewInt(987), big.NewInt(65_432))
	rpc := &quoteHTTPRPCClient{response: response}
	poolKey := testPoolKey(2)
	quoter := common.HexToAddress("0x0000000000000000000000000000000000000005")
	client := quoteTestClient(rpc, poolKey, quoter)
	exactAmount := big.NewInt(123_456)
	hookData := []byte{0xaa, 0xbb, 0xcc}
	if blockNumber == nil {
		hookData = nil
	}

	var amount, gas *big.Int
	var signature string
	var err error
	if exactInput {
		result, quoteErr := client.QuoteExactInputSingle(context.Background(), QuoteExactInputSingleParams{
			PoolKey: poolKey, ZeroForOne: zeroForOne, AmountIn: exactAmount, HookData: hookData,
		}, blockNumber)
		amount, gas, signature, err = result.AmountOut, result.GasEstimate, quoteExactInputSingleSignature, quoteErr
	} else {
		result, quoteErr := client.QuoteExactOutputSingle(context.Background(), QuoteExactOutputSingleParams{
			PoolKey: poolKey, ZeroForOne: zeroForOne, AmountOut: exactAmount, HookData: hookData,
		}, blockNumber)
		amount, gas, signature, err = result.AmountIn, result.GasEstimate, quoteExactOutputSingleSignature, quoteErr
	}
	if err != nil {
		t.Fatalf("quote error = %v", err)
	}
	if amount.Cmp(big.NewInt(987)) != 0 || gas.Cmp(big.NewInt(65_432)) != 0 {
		t.Fatalf("result = (%s, %s), want (987, 65432)", amount, gas)
	}
	if rpc.call.To == nil || *rpc.call.To != quoter {
		t.Fatalf("CallMsg.To = %v, want %s", rpc.call.To, quoter.Hex())
	}
	if rpc.blockNumber != blockNumber {
		t.Fatal("block number pointer was not propagated")
	}
	if !bytes.Equal(rpc.call.Data[:4], crypto.Keccak256([]byte(signature))[:4]) {
		t.Fatalf("selector = %x, want signature %s", rpc.call.Data[:4], signature)
	}
	decoded := unpackQuoteParams(t, rpc.call.Data[4:])
	if decoded.PoolKey.Currency0 != poolKey.Currency0.Address() || decoded.PoolKey.Currency1 != poolKey.Currency1.Address() ||
		decoded.PoolKey.Fee.Uint64() != uint64(poolKey.Fee) || decoded.PoolKey.TickSpacing.Int64() != int64(poolKey.TickSpacing) ||
		decoded.PoolKey.Hooks != poolKey.Hooks || decoded.ZeroForOne != zeroForOne || decoded.ExactAmount.Cmp(exactAmount) != 0 ||
		!bytes.Equal(decoded.HookData, hookData) {
		t.Fatalf("decoded params = %+v", decoded)
	}
}

func TestQuoteExactSingleValidation(t *testing.T) {
	t.Parallel()
	poolKey := testPoolKey(2)
	configuredClient := quoteTestClient(&quoteHTTPRPCClient{response: packQuoteResult(t, big.NewInt(1), big.NewInt(0))}, poolKey, common.HexToAddress("0x5"))
	overflow := new(big.Int).Lsh(big.NewInt(1), 128)

	tests := []struct {
		name        string
		client      *HTTPClient
		poolKey     protocol.PoolKey
		amount      *big.Int
		blockNumber *big.Int
		want        string
	}{
		{"nil client", nil, poolKey, big.NewInt(1), nil, "client=null"},
		{"nil rpc", &HTTPClient{quoter: common.HexToAddress("0x5"), poolKeys: []protocol.PoolKey{poolKey}}, poolKey, big.NewInt(1), nil, "http_rpc_client=null"},
		{"empty quoter", &HTTPClient{rpc: &quoteHTTPRPCClient{}, poolKeys: []protocol.PoolKey{poolKey}}, poolKey, big.NewInt(1), nil, "quoter=empty"},
		{"invalid pool key", configuredClient, protocol.PoolKey{}, big.NewInt(1), nil, "currency_order=invalid"},
		{"unconfigured pool key", configuredClient, testPoolKey(3), big.NewInt(1), nil, "pool_id=unconfigured"},
		{"nil amount", configuredClient, poolKey, nil, nil, "amount_in=null"},
		{"zero amount", configuredClient, poolKey, big.NewInt(0), nil, "amount_in=out_of_range"},
		{"negative amount", configuredClient, poolKey, big.NewInt(-1), nil, "amount_in=out_of_range"},
		{"overflow amount", configuredClient, poolKey, overflow, nil, "amount_in=out_of_range"},
		{"negative block", configuredClient, poolKey, big.NewInt(1), big.NewInt(-1), "block_number=out_of_range"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := test.client.QuoteExactInputSingle(context.Background(), QuoteExactInputSingleParams{PoolKey: test.poolKey, AmountIn: test.amount}, test.blockNumber)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestQuoteExactOutputSingleAmountValidation(t *testing.T) {
	t.Parallel()
	poolKey := testPoolKey(2)
	client := quoteTestClient(&quoteHTTPRPCClient{}, poolKey, common.HexToAddress("0x5"))
	for _, amount := range []*big.Int{nil, big.NewInt(0), big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 128)} {
		_, err := client.QuoteExactOutputSingle(context.Background(), QuoteExactOutputSingleParams{PoolKey: poolKey, AmountOut: amount}, nil)
		want := "amount_out=out_of_range"
		if amount == nil {
			want = "amount_out=null"
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want containing %q", err, want)
		}
	}
}

func TestQuoteExactSingleRejectsInvalidResponse(t *testing.T) {
	t.Parallel()
	poolKey := testPoolKey(2)
	tests := []struct {
		name     string
		response []byte
		want     string
	}{
		{"empty", nil, "response=empty"},
		{"malformed", []byte{1}, "failed to decode"},
		{"zero amount", packQuoteResult(t, big.NewInt(0), big.NewInt(1)), "amount_out=invalid"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			client := quoteTestClient(&quoteHTTPRPCClient{response: test.response}, poolKey, common.HexToAddress("0x5"))
			_, err := client.QuoteExactInputSingle(context.Background(), QuoteExactInputSingleParams{PoolKey: poolKey, AmountIn: big.NewInt(1)}, nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestQuoteExactSingleWrapsRPCError(t *testing.T) {
	t.Parallel()
	underlying := errors.New("rpc failed")
	poolKey := testPoolKey(2)
	client := quoteTestClient(&quoteHTTPRPCClient{err: underlying}, poolKey, common.HexToAddress("0x5"))
	_, err := client.QuoteExactInputSingle(context.Background(), QuoteExactInputSingleParams{PoolKey: poolKey, AmountIn: big.NewInt(1)}, nil)
	if !errors.Is(err, underlying) {
		t.Fatalf("error = %v, want wrapped RPC error", err)
	}
	_, err = client.QuoteExactOutputSingle(context.Background(), QuoteExactOutputSingleParams{PoolKey: poolKey, AmountOut: big.NewInt(1)}, nil)
	if !errors.Is(err, underlying) {
		t.Fatalf("exact-output error = %v, want wrapped RPC error", err)
	}
}

func TestQuoteExactSingleSignatures(t *testing.T) {
	t.Parallel()

	if quoteExactInputSingleSignature != "quoteExactInputSingle(((address,address,uint24,int24,address),bool,uint128,bytes))" {
		t.Fatalf("exact-input signature = %q", quoteExactInputSingleSignature)
	}
	if quoteExactOutputSingleSignature != "quoteExactOutputSingle(((address,address,uint24,int24,address),bool,uint128,bytes))" {
		t.Fatalf("exact-output signature = %q", quoteExactOutputSingleSignature)
	}
}

func quoteTestClient(rpc HTTPRPCClient, poolKey protocol.PoolKey, quoter common.Address) *HTTPClient {
	return &HTTPClient{rpc: rpc, quoter: quoter, poolKeys: []protocol.PoolKey{poolKey}}
}

func packQuoteResult(t *testing.T, amount, gas *big.Int) []byte {
	t.Helper()
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := (abi.Arguments{{Type: uint256Type}, {Type: uint256Type}}).Pack(amount, gas)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func unpackQuoteParams(t *testing.T, data []byte) quoteExactSingleParamsABI {
	t.Helper()
	paramsType, err := quoteExactSingleParamsType()
	if err != nil {
		t.Fatal(err)
	}
	values, err := (abi.Arguments{{Type: paramsType}}).Unpack(data)
	if err != nil {
		t.Fatal(err)
	}
	converted := abi.ConvertType(values[0], new(quoteExactSingleParamsABI))
	params, ok := converted.(*quoteExactSingleParamsABI)
	if !ok {
		t.Fatalf("decoded type = %T", converted)
	}
	return *params
}
