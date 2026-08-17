package protocol

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

const (
	MaxStaticLPFee uint32 = 1_000_000
	DynamicFeeFlag uint32 = 0x800000

	MinTickSpacing int32 = 1
	MaxTickSpacing int32 = 32_767
)

const (
	allHookMask                          uint16 = (1 << 14) - 1
	afterAddLiquidityHookFlag            uint16 = 1 << 10
	afterRemoveLiquidityHookFlag         uint16 = 1 << 8
	beforeSwapHookFlag                   uint16 = 1 << 7
	afterSwapHookFlag                    uint16 = 1 << 6
	beforeSwapReturnsDeltaHookFlag       uint16 = 1 << 3
	afterSwapReturnsDeltaHookFlag        uint16 = 1 << 2
	afterAddLiquidityReturnsDeltaFlag    uint16 = 1 << 1
	afterRemoveLiquidityReturnsDeltaFlag uint16 = 1 << 0
)

type PoolKey struct {
	Currency0   Currency
	Currency1   Currency
	Fee         uint32
	TickSpacing int32
	Hooks       common.Address
}

// Validate validates a Uniswap v4 pool key.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-17: Added.
func (k PoolKey) Validate() error {
	currency0 := k.Currency0.Address()
	currency1 := k.Currency1.Address()
	if bytes.Compare(currency0[:], currency1[:]) >= 0 {
		return fmt.Errorf("failed to validate uniswap v4 pool key: currency_order=invalid")
	}

	if k.Fee > MaxStaticLPFee && k.Fee != DynamicFeeFlag {
		return fmt.Errorf(
			"failed to validate uniswap v4 pool key: fee=out_of_range max_static_fee=%d dynamic_fee_flag=%d",
			MaxStaticLPFee,
			DynamicFeeFlag,
		)
	}

	if k.TickSpacing < MinTickSpacing || k.TickSpacing > MaxTickSpacing {
		return fmt.Errorf(
			"failed to validate uniswap v4 pool key: tick_spacing=out_of_range min_value=%d max_value=%d",
			MinTickSpacing,
			MaxTickSpacing,
		)
	}

	if !isValidHookAddress(k.Hooks, k.Fee) {
		return fmt.Errorf("failed to validate uniswap v4 pool key: hooks=invalid")
	}

	return nil
}

func isValidHookAddress(hooks common.Address, fee uint32) bool {
	flags := hookFlags(hooks)
	if flags&beforeSwapHookFlag == 0 && flags&beforeSwapReturnsDeltaHookFlag != 0 {
		return false
	}
	if flags&afterSwapHookFlag == 0 && flags&afterSwapReturnsDeltaHookFlag != 0 {
		return false
	}
	if flags&afterAddLiquidityHookFlag == 0 && flags&afterAddLiquidityReturnsDeltaFlag != 0 {
		return false
	}
	if flags&afterRemoveLiquidityHookFlag == 0 && flags&afterRemoveLiquidityReturnsDeltaFlag != 0 {
		return false
	}

	dynamicFee := fee == DynamicFeeFlag
	if hooks == (common.Address{}) {
		return !dynamicFee
	}

	return flags != 0 || dynamicFee
}

func hookFlags(hooks common.Address) uint16 {
	return (uint16(hooks[18])<<8 | uint16(hooks[19])) & allHookMask
}
