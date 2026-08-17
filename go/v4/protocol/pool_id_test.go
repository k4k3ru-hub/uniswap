package protocol

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNewPoolID(t *testing.T) {
	hash := common.HexToHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	poolID := NewPoolID(hash)

	if poolID.Hash() != hash {
		t.Fatalf("Hash() = %s, want %s", poolID.Hash().Hex(), hash.Hex())
	}
	if poolID.Hex() != hash.Hex() {
		t.Fatalf("Hex() = %q, want %q", poolID.Hex(), hash.Hex())
	}
	if poolID.IsZero() {
		t.Fatal("IsZero() = true, want false")
	}
}

func TestParsePoolID(t *testing.T) {
	value := "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	want := common.HexToHash(value)

	poolID, err := ParsePoolID(value)
	if err != nil {
		t.Fatalf("ParsePoolID() error = %v", err)
	}
	if poolID.Hash() != want {
		t.Fatalf("Hash() = %s, want %s", poolID.Hash().Hex(), want.Hex())
	}
}

func TestParsePoolIDWrapsHashParseError(t *testing.T) {
	_, err := ParsePoolID("")
	if err == nil {
		t.Fatal("ParsePoolID() error = nil")
	}
	if !strings.Contains(err.Error(), "failed to parse uniswap v4 pool id: failed to parse evm hash") {
		t.Fatalf("ParsePoolID() error = %q, want wrapped EVM hash parse error", err)
	}
}

func TestPoolIDIsZero(t *testing.T) {
	if !NewPoolID(common.Hash{}).IsZero() {
		t.Fatal("IsZero() = false, want true")
	}
}

func TestPoolKeyIDMatchesKnownPoolID(t *testing.T) {
	key := PoolKey{
		Currency0:   NewCurrency(common.HexToAddress("0x5991A2dF15A8F6A256D3Ec51E99254Cd3fb576A9")),
		Currency1:   NewCurrency(common.HexToAddress("0xF62849F9A0B5Bf2913b396098F7c7019b51A820a")),
		Fee:         3_000,
		TickSpacing: 60,
		Hooks:       common.HexToAddress("0x0000000000000000000000000000000000003000"),
	}
	want := "0x3028868e330056d8b8eb33861acc05f54f6cc1d2f217f4f0dc9490a9deb5f917"

	poolID, err := key.ID()
	if err != nil {
		t.Fatalf("ID() error = %v", err)
	}
	if poolID.Hex() != want {
		t.Fatalf("ID() = %s, want %s", poolID.Hex(), want)
	}
}

func TestPoolKeyIDRejectsInvalidPoolKey(t *testing.T) {
	key := validPoolKey()
	key.Currency1 = key.Currency0

	_, err := key.ID()
	if err == nil {
		t.Fatal("ID() error = nil")
	}
	if !strings.Contains(err.Error(), "failed to calculate uniswap v4 pool id: failed to validate uniswap v4 pool key") {
		t.Fatalf("ID() error = %q, want wrapped pool key validation error", err)
	}
}

func TestPoolKeyIDChangesWhenPoolKeyChanges(t *testing.T) {
	original := validPoolKey()
	originalID, err := original.ID()
	if err != nil {
		t.Fatalf("ID() error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*PoolKey)
	}{
		{name: "fee", mutate: func(key *PoolKey) { key.Fee = 500 }},
		{name: "tick spacing", mutate: func(key *PoolKey) { key.TickSpacing = 10 }},
		{
			name: "hooks",
			mutate: func(key *PoolKey) {
				key.Hooks = common.HexToAddress("0x0000000000000000000000000000000000000080")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := original
			test.mutate(&key)
			poolID, idErr := key.ID()
			if idErr != nil {
				t.Fatalf("ID() error = %v", idErr)
			}
			if poolID == originalID {
				t.Fatalf("ID() = %s, want value different from original", poolID.Hex())
			}
		})
	}
}
