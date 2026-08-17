package v4

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/k4k3ru-hub/uniswap/go/v4/protocol"
)

const swapEventSignature = "Swap(bytes32,address,int128,int128,uint160,uint128,int24,uint24)"

type Swap struct {
	PoolManager common.Address
	PoolID      protocol.PoolID
	Sender      common.Address

	Amount0      *big.Int
	Amount1      *big.Int
	SqrtPriceX96 *big.Int
	Liquidity    *big.Int
	Tick         int32
	Fee          uint32

	BlockNumber      uint64
	BlockHash        common.Hash
	TransactionHash  common.Hash
	TransactionIndex uint
	LogIndex         uint
	Removed          bool
}

// DecodeSwapLog decodes a Uniswap v4 PoolManager Swap event log.
//
// Parameters:
//   - eventLog: EVM event log.
//
// Returns:
//   - Decoded Swap event.
//   - Decode error.
//
// Version:
//   - 2026-08-17: Added.
func DecodeSwapLog(eventLog types.Log) (Swap, error) {
	if len(eventLog.Topics) != 3 {
		return Swap{}, fmt.Errorf(
			"failed to decode uniswap v4 swap log: topics=invalid actual_length=%d expected_length=3",
			len(eventLog.Topics),
		)
	}
	if eventLog.Topics[0] != swapEventSignatureHash() {
		return Swap{}, fmt.Errorf("failed to decode uniswap v4 swap log: event_signature=invalid")
	}

	poolID := protocol.NewPoolID(eventLog.Topics[1])
	if poolID.IsZero() {
		return Swap{}, fmt.Errorf("failed to decode uniswap v4 swap log: pool_id=empty")
	}
	if !bytes.Equal(eventLog.Topics[2][:12], make([]byte, 12)) {
		return Swap{}, fmt.Errorf("failed to decode uniswap v4 swap log: sender_topic=invalid")
	}
	sender := common.BytesToAddress(eventLog.Topics[2][12:])

	amount0, amount1, sqrtPriceX96, liquidity, tick, fee, err := decodeSwapData(eventLog.Data)
	if err != nil {
		return Swap{}, fmt.Errorf("failed to decode uniswap v4 swap log: %w", err)
	}

	return Swap{
		PoolManager:      eventLog.Address,
		PoolID:           poolID,
		Sender:           sender,
		Amount0:          amount0,
		Amount1:          amount1,
		SqrtPriceX96:     sqrtPriceX96,
		Liquidity:        liquidity,
		Tick:             tick,
		Fee:              fee,
		BlockNumber:      eventLog.BlockNumber,
		BlockHash:        eventLog.BlockHash,
		TransactionHash:  eventLog.TxHash,
		TransactionIndex: eventLog.TxIndex,
		LogIndex:         eventLog.Index,
		Removed:          eventLog.Removed,
	}, nil
}

func swapEventSignatureHash() common.Hash {
	return crypto.Keccak256Hash([]byte(swapEventSignature))
}

func decodeSwapData(data []byte) (*big.Int, *big.Int, *big.Int, *big.Int, int32, uint32, error) {
	const (
		wordLength        = 32
		expectedWordCount = 6
	)
	if len(data) != wordLength*expectedWordCount {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf(
			"failed to decode swap event data: data=invalid actual_length=%d expected_length=%d",
			len(data),
			wordLength*expectedWordCount,
		)
	}
	if !hasCanonicalSignedPadding(data[0:32], 128) {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: amount0=invalid")
	}
	if !hasCanonicalSignedPadding(data[32:64], 128) {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: amount1=invalid")
	}
	if !hasCanonicalUnsignedPadding(data[64:96], 160) {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: sqrt_price_x96=invalid")
	}
	if !hasCanonicalUnsignedPadding(data[96:128], 128) {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: liquidity=invalid")
	}
	if !hasCanonicalSignedPadding(data[128:160], 24) {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: tick=invalid")
	}
	if !hasCanonicalUnsignedPadding(data[160:192], 24) {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: fee=invalid")
	}

	arguments, err := swapDataArguments()
	if err != nil {
		return nil, nil, nil, nil, 0, 0, err
	}
	values, err := arguments.Unpack(data)
	if err != nil {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: %w", err)
	}
	if len(values) != expectedWordCount {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf(
			"failed to decode swap event data: values=invalid actual_length=%d expected_length=%d",
			len(values),
			expectedWordCount,
		)
	}

	amount0, ok := values[0].(*big.Int)
	if !ok {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: amount0=invalid")
	}
	amount1, ok := values[1].(*big.Int)
	if !ok {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: amount1=invalid")
	}
	sqrtPriceX96, ok := values[2].(*big.Int)
	if !ok {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: sqrt_price_x96=invalid")
	}
	liquidity, ok := values[3].(*big.Int)
	if !ok {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: liquidity=invalid")
	}
	tick, ok := values[4].(*big.Int)
	if !ok || !tick.IsInt64() {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: tick=invalid")
	}
	fee, ok := values[5].(*big.Int)
	if !ok || !fee.IsUint64() {
		return nil, nil, nil, nil, 0, 0, fmt.Errorf("failed to decode swap event data: fee=invalid")
	}

	return new(big.Int).Set(amount0),
		new(big.Int).Set(amount1),
		new(big.Int).Set(sqrtPriceX96),
		new(big.Int).Set(liquidity),
		int32(tick.Int64()),
		uint32(fee.Uint64()),
		nil
}

func swapDataArguments() (abi.Arguments, error) {
	typeNames := []string{"int128", "int128", "uint160", "uint128", "int24", "uint24"}
	arguments := make(abi.Arguments, 0, len(typeNames))
	for _, typeName := range typeNames {
		argumentType, err := abi.NewType(typeName, "", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create swap event data arguments: %w: abi_type=%q", err, typeName)
		}
		arguments = append(arguments, abi.Argument{Type: argumentType})
	}

	return arguments, nil
}

func hasCanonicalUnsignedPadding(word []byte, bitSize int) bool {
	valueByteLength := bitSize / 8
	return len(word) == 32 && bytes.Equal(word[:len(word)-valueByteLength], make([]byte, len(word)-valueByteLength))
}

func hasCanonicalSignedPadding(word []byte, bitSize int) bool {
	if len(word) != 32 {
		return false
	}
	valueStart := len(word) - bitSize/8
	paddingByte := byte(0)
	if word[valueStart]&0x80 != 0 {
		paddingByte = 0xff
	}
	for _, value := range word[:valueStart] {
		if value != paddingByte {
			return false
		}
	}
	return true
}
