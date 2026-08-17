package v4

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestDecodeSwapLog(t *testing.T) {
	t.Parallel()

	poolID := common.HexToHash("0x1234")
	sender := common.HexToAddress("0x0000000000000000000000000000000000005678")
	poolManager := common.HexToAddress("0x0000000000000000000000000000000000009999")
	amount0 := big.NewInt(-100_000_000)
	amount1 := big.NewInt(6_500_000_000)
	sqrtPriceX96 := new(big.Int).Lsh(big.NewInt(1), 96)
	liquidity := new(big.Int).Lsh(big.NewInt(1), 100)

	eventLog := types.Log{
		Address: poolManager,
		Topics: []common.Hash{
			crypto.Keccak256Hash([]byte(swapEventSignature)),
			poolID,
			common.BytesToHash(sender.Bytes()),
		},
		Data:        encodeSwapData(t, amount0, amount1, sqrtPriceX96, liquidity, -139, 3_000),
		BlockNumber: 12_345,
		BlockHash:   common.HexToHash("0xabcd"),
		TxHash:      common.HexToHash("0x9876"),
		TxIndex:     7,
		Index:       11,
		Removed:     true,
	}

	got, err := DecodeSwapLog(eventLog)
	if err != nil {
		t.Fatalf("DecodeSwapLog() error = %v", err)
	}
	if got.PoolManager != poolManager || got.PoolID.Hash() != poolID || got.Sender != sender {
		t.Fatalf("decoded identities = manager:%s pool:%s sender:%s", got.PoolManager.Hex(), got.PoolID.Hex(), got.Sender.Hex())
	}
	if got.Amount0.Cmp(amount0) != 0 || got.Amount1.Cmp(amount1) != 0 {
		t.Fatalf("decoded amounts = (%s, %s), want (%s, %s)", got.Amount0, got.Amount1, amount0, amount1)
	}
	if got.SqrtPriceX96.Cmp(sqrtPriceX96) != 0 || got.Liquidity.Cmp(liquidity) != 0 {
		t.Fatalf("decoded state = price:%s liquidity:%s", got.SqrtPriceX96, got.Liquidity)
	}
	if got.Tick != -139 || got.Fee != 3_000 {
		t.Fatalf("decoded tick and fee = (%d, %d), want (-139, 3000)", got.Tick, got.Fee)
	}
	if got.BlockNumber != eventLog.BlockNumber || got.BlockHash != eventLog.BlockHash || got.TransactionHash != eventLog.TxHash || got.TransactionIndex != eventLog.TxIndex || got.LogIndex != eventLog.Index || got.Removed != eventLog.Removed {
		t.Fatalf("decoded log metadata = %+v", got)
	}
}

func TestDecodeSwapLogRejectsInvalidLog(t *testing.T) {
	t.Parallel()

	validLog := newSwapLog(t)
	tests := []struct {
		name      string
		mutate    func(*types.Log)
		wantError string
	}{
		{name: "missing topics", mutate: func(log *types.Log) { log.Topics = nil }, wantError: "topics=invalid"},
		{name: "extra topic", mutate: func(log *types.Log) { log.Topics = append(log.Topics, common.Hash{}) }, wantError: "topics=invalid"},
		{name: "wrong signature", mutate: func(log *types.Log) { log.Topics[0] = common.HexToHash("0x01") }, wantError: "event_signature=invalid"},
		{name: "empty pool ID", mutate: func(log *types.Log) { log.Topics[1] = common.Hash{} }, wantError: "pool_id=empty"},
		{name: "invalid sender padding", mutate: func(log *types.Log) { log.Topics[2][0] = 1 }, wantError: "sender_topic=invalid"},
		{name: "short data", mutate: func(log *types.Log) { log.Data = log.Data[:191] }, wantError: "data=invalid"},
		{name: "long data", mutate: func(log *types.Log) { log.Data = append(log.Data, 0) }, wantError: "data=invalid"},
		{name: "invalid amount0 padding", mutate: func(log *types.Log) { log.Data[0] = 1 }, wantError: "amount0=invalid"},
		{name: "invalid amount1 padding", mutate: func(log *types.Log) { log.Data[32] = 1 }, wantError: "amount1=invalid"},
		{name: "invalid sqrt price padding", mutate: func(log *types.Log) { log.Data[64] = 1 }, wantError: "sqrt_price_x96=invalid"},
		{name: "invalid liquidity padding", mutate: func(log *types.Log) { log.Data[96] = 1 }, wantError: "liquidity=invalid"},
		{name: "invalid tick padding", mutate: func(log *types.Log) { log.Data[128] = 1 }, wantError: "tick=invalid"},
		{name: "invalid fee padding", mutate: func(log *types.Log) { log.Data[160] = 1 }, wantError: "fee=invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			eventLog := cloneLog(validLog)
			test.mutate(&eventLog)
			_, err := DecodeSwapLog(eventLog)
			if err == nil {
				t.Fatal("DecodeSwapLog() error = nil")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("DecodeSwapLog() error = %q, want substring %q", err, test.wantError)
			}
		})
	}
}

func newSwapLog(t *testing.T) types.Log {
	t.Helper()

	return types.Log{
		Address: common.HexToAddress("0x0000000000000000000000000000000000009999"),
		Topics: []common.Hash{
			crypto.Keccak256Hash([]byte(swapEventSignature)),
			common.HexToHash("0x1234"),
			common.BytesToHash(common.HexToAddress("0x0000000000000000000000000000000000005678").Bytes()),
		},
		Data: encodeSwapData(t, big.NewInt(-1), big.NewInt(2), big.NewInt(3), big.NewInt(4), -5, 6),
	}
}

func cloneLog(eventLog types.Log) types.Log {
	cloned := eventLog
	cloned.Topics = append([]common.Hash(nil), eventLog.Topics...)
	cloned.Data = append([]byte(nil), eventLog.Data...)
	return cloned
}

func encodeSwapData(t *testing.T, amount0, amount1, sqrtPriceX96, liquidity *big.Int, tick int32, fee uint32) []byte {
	t.Helper()

	typeNames := []string{"int128", "int128", "uint160", "uint128", "int24", "uint24"}
	arguments := make(abi.Arguments, 0, len(typeNames))
	for _, typeName := range typeNames {
		argumentType, err := abi.NewType(typeName, "", nil)
		if err != nil {
			t.Fatalf("abi.NewType(%s) error = %v", typeName, err)
		}
		arguments = append(arguments, abi.Argument{Type: argumentType})
	}

	data, err := arguments.Pack(
		amount0,
		amount1,
		sqrtPriceX96,
		liquidity,
		big.NewInt(int64(tick)),
		new(big.Int).SetUint64(uint64(fee)),
	)
	if err != nil {
		t.Fatalf("Arguments.Pack() error = %v", err)
	}
	return data
}
