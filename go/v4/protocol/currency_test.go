package protocol

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNewCurrency(t *testing.T) {
	address := common.HexToAddress("0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	currency := NewCurrency(address)

	if currency.Address() != address {
		t.Fatalf("Address() = %s, want %s", currency.Address().Hex(), address.Hex())
	}
	if currency.Hex() != address.Hex() {
		t.Fatalf("Hex() = %q, want %q", currency.Hex(), address.Hex())
	}
	if currency.IsNative() {
		t.Fatal("IsNative() = true, want false")
	}
	if !currency.IsERC20() {
		t.Fatal("IsERC20() = false, want true")
	}
}

func TestParseCurrency(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  common.Address
	}{
		{
			name:  "checksummed address",
			value: "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eB48",
			want:  common.HexToAddress("0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
		},
		{
			name:  "lowercase address",
			value: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			want:  common.HexToAddress("0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
		},
		{
			name:  "native currency",
			value: "0x0000000000000000000000000000000000000000",
			want:  common.Address{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			currency, err := ParseCurrency(test.value)
			if err != nil {
				t.Fatalf("ParseCurrency() error = %v", err)
			}
			if currency.Address() != test.want {
				t.Fatalf("Address() = %s, want %s", currency.Address().Hex(), test.want.Hex())
			}
		})
	}
}

func TestParseCurrencyRejectsInvalidAddress(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "invalid hexadecimal", value: "0xinvalid"},
		{name: "too long", value: "0x" + strings.Repeat("1", 41)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseCurrency(test.value)
			if err == nil {
				t.Fatal("ParseCurrency() error = nil")
			}
			if !strings.Contains(err.Error(), "failed to parse uniswap v4 currency: failed to parse evm address") {
				t.Fatalf("ParseCurrency() error = %q, want wrapped onchain parse error", err)
			}
		})
	}
}

func TestCurrencyNativeState(t *testing.T) {
	currency := NewCurrency(common.Address{})

	if !currency.IsNative() {
		t.Fatal("IsNative() = false, want true")
	}
	if currency.IsERC20() {
		t.Fatal("IsERC20() = true, want false")
	}
}
