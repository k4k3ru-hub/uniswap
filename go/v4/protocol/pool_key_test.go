package protocol

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func validPoolKey() PoolKey {
	return PoolKey{
		Currency0:   NewCurrency(common.HexToAddress("0x0000000000000000000000000000000000000001")),
		Currency1:   NewCurrency(common.HexToAddress("0x0000000000000000000000000000000000000002")),
		Fee:         3_000,
		TickSpacing: 60,
	}
}

func TestPoolKeyValidateAcceptsValidPoolKeys(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*PoolKey)
	}{
		{name: "static fee without hooks", mutate: func(*PoolKey) {}},
		{
			name: "static fee with hook permission",
			mutate: func(key *PoolKey) {
				key.Hooks = common.HexToAddress("0x0000000000000000000000000000000000000080")
			},
		},
		{
			name: "dynamic fee with hooks",
			mutate: func(key *PoolKey) {
				key.Fee = DynamicFeeFlag
				key.Hooks = common.HexToAddress("0x0000000000000000000000000000000000004000")
			},
		},
		{
			name: "maximum static fee",
			mutate: func(key *PoolKey) {
				key.Fee = MaxStaticLPFee
			},
		},
		{
			name: "minimum tick spacing",
			mutate: func(key *PoolKey) {
				key.TickSpacing = MinTickSpacing
			},
		},
		{
			name: "maximum tick spacing",
			mutate: func(key *PoolKey) {
				key.TickSpacing = MaxTickSpacing
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := validPoolKey()
			test.mutate(&key)
			if err := key.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestPoolKeyValidateRejectsInvalidPoolKeys(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*PoolKey)
	}{
		{
			name: "equal currencies",
			mutate: func(key *PoolKey) {
				key.Currency1 = key.Currency0
			},
		},
		{
			name: "reversed currencies",
			mutate: func(key *PoolKey) {
				key.Currency0, key.Currency1 = key.Currency1, key.Currency0
			},
		},
		{
			name: "static fee above maximum",
			mutate: func(key *PoolKey) {
				key.Fee = MaxStaticLPFee + 1
			},
		},
		{
			name: "fee above uint24",
			mutate: func(key *PoolKey) {
				key.Fee = 1 << 24
			},
		},
		{
			name: "tick spacing below minimum",
			mutate: func(key *PoolKey) {
				key.TickSpacing = MinTickSpacing - 1
			},
		},
		{
			name: "tick spacing above maximum",
			mutate: func(key *PoolKey) {
				key.TickSpacing = MaxTickSpacing + 1
			},
		},
		{
			name: "dynamic fee without hooks",
			mutate: func(key *PoolKey) {
				key.Fee = DynamicFeeFlag
			},
		},
		{
			name: "static fee with no hook permissions",
			mutate: func(key *PoolKey) {
				key.Hooks = common.HexToAddress("0x0000000000000000000000000000000000004000")
			},
		},
		{
			name: "before swap return delta without before swap",
			mutate: func(key *PoolKey) {
				key.Hooks = common.HexToAddress("0x0000000000000000000000000000000000000008")
			},
		},
		{
			name: "after swap return delta without after swap",
			mutate: func(key *PoolKey) {
				key.Hooks = common.HexToAddress("0x0000000000000000000000000000000000000004")
			},
		},
		{
			name: "after add liquidity return delta without after add liquidity",
			mutate: func(key *PoolKey) {
				key.Hooks = common.HexToAddress("0x0000000000000000000000000000000000000002")
			},
		},
		{
			name: "after remove liquidity return delta without after remove liquidity",
			mutate: func(key *PoolKey) {
				key.Hooks = common.HexToAddress("0x0000000000000000000000000000000000000001")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := validPoolKey()
			test.mutate(&key)
			if err := key.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}
