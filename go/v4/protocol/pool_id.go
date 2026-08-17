package protocol

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	myOnchainEVM "github.com/k4k3ru-hub/onchain/go/evm"
)

type PoolID common.Hash

// NewPoolID creates a Uniswap v4 pool ID.
//
// Parameters:
//   - hash: EVM hash.
//
// Returns:
//   - Uniswap v4 pool ID.
//
// Version:
//   - 2026-08-17: Added.
func NewPoolID(hash common.Hash) PoolID {
	return PoolID(hash)
}

// ParsePoolID parses a complete 32-byte EVM hash as a Uniswap v4 pool ID.
//
// Parameters:
//   - value: 0x-prefixed hexadecimal pool ID.
//
// Returns:
//   - Parsed pool ID.
//   - Parse error.
//
// Version:
//   - 2026-08-17: Added.
func ParsePoolID(value string) (PoolID, error) {
	hash, err := myOnchainEVM.ParseHash(value)
	if err != nil {
		return PoolID{}, fmt.Errorf("failed to parse uniswap v4 pool id: %w", err)
	}

	return NewPoolID(hash), nil
}

// Hash returns the underlying EVM hash.
//
// Returns:
//   - EVM hash.
//
// Version:
//   - 2026-08-17: Added.
func (id PoolID) Hash() common.Hash {
	return common.Hash(id)
}

// Hex returns the hexadecimal pool ID.
//
// Returns:
//   - 0x-prefixed hexadecimal pool ID.
//
// Version:
//   - 2026-08-17: Added.
func (id PoolID) Hex() string {
	return id.Hash().Hex()
}

// IsZero reports whether the pool ID is zero.
//
// Returns:
//   - True when the pool ID is zero.
//
// Version:
//   - 2026-08-17: Added.
func (id PoolID) IsZero() bool {
	return id.Hash() == (common.Hash{})
}

// ID calculates the Uniswap v4 pool ID.
//
// Returns:
//   - Pool ID.
//   - Calculation error.
//
// Version:
//   - 2026-08-17: Added.
func (k PoolKey) ID() (PoolID, error) {
	if err := k.Validate(); err != nil {
		return PoolID{}, fmt.Errorf("failed to calculate uniswap v4 pool id: %w", err)
	}

	encoded, err := encodePoolKey(k)
	if err != nil {
		return PoolID{}, fmt.Errorf("failed to calculate uniswap v4 pool id: %w", err)
	}

	return NewPoolID(crypto.Keccak256Hash(encoded)), nil
}

func encodePoolKey(key PoolKey) ([]byte, error) {
	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encode uniswap v4 pool key: %w: abi_type=%q", err, "address")
	}
	uint24Type, err := abi.NewType("uint24", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encode uniswap v4 pool key: %w: abi_type=%q", err, "uint24")
	}
	int24Type, err := abi.NewType("int24", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encode uniswap v4 pool key: %w: abi_type=%q", err, "int24")
	}

	arguments := abi.Arguments{
		{Type: addressType},
		{Type: addressType},
		{Type: uint24Type},
		{Type: int24Type},
		{Type: addressType},
	}
	encoded, err := arguments.Pack(
		key.Currency0.Address(),
		key.Currency1.Address(),
		new(big.Int).SetUint64(uint64(key.Fee)),
		big.NewInt(int64(key.TickSpacing)),
		key.Hooks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode uniswap v4 pool key: %w", err)
	}

	return encoded, nil
}
