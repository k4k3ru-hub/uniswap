package protocol

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	myOnchainEVM "github.com/k4k3ru-hub/onchain/go/evm"
)

type Currency common.Address

// NewCurrency creates a Uniswap v4 currency.
//
// Parameters:
//   - address: EVM address.
//
// Returns:
//   - Uniswap v4 currency.
//
// Version:
//   - 2026-08-17: Added.
func NewCurrency(address common.Address) Currency {
	return Currency(address)
}

// ParseCurrency parses an EVM address as a Uniswap v4 currency.
//
// Parameters:
//   - value: Currency address.
//
// Returns:
//   - Parsed currency.
//   - Parse error.
//
// Version:
//   - 2026-08-17: Added.
func ParseCurrency(value string) (Currency, error) {
	address, err := myOnchainEVM.ParseAddress(value)
	if err != nil {
		return Currency{}, fmt.Errorf("failed to parse uniswap v4 currency: %w", err)
	}

	return NewCurrency(address), nil
}

// Address returns the underlying EVM address.
//
// Returns:
//   - EVM address.
//
// Version:
//   - 2026-08-17: Added.
func (c Currency) Address() common.Address {
	return common.Address(c)
}

// Hex returns the checksummed hexadecimal address.
//
// Returns:
//   - Checksummed hexadecimal address.
//
// Version:
//   - 2026-08-17: Added.
func (c Currency) Hex() string {
	return c.Address().Hex()
}

// IsNative reports whether the currency represents native currency.
//
// Returns:
//   - True when the currency is native.
//
// Version:
//   - 2026-08-17: Added.
func (c Currency) IsNative() bool {
	return c.Address() == (common.Address{})
}

// IsERC20 reports whether the currency represents an ERC-20 token.
//
// Returns:
//   - True when the currency represents an ERC-20 token.
//
// Version:
//   - 2026-08-17: Added.
func (c Currency) IsERC20() bool {
	return !c.IsNative()
}
