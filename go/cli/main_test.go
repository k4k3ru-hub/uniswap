package main

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	myUniswapV4 "github.com/k4k3ru-hub/uniswap/go/v4"
)

func TestGetSlot0Command(t *testing.T) {
	t.Parallel()

	var received getSlot0Input
	applicationCLI, err := newCLI(func(_ context.Context, input getSlot0Input) (getSlot0Output, error) {
		received = input
		poolID, err := input.PoolKey.ID()
		if err != nil {
			return getSlot0Output{}, err
		}
		return getSlot0Output{
			PoolID: poolID,
			Slot0: myUniswapV4.Slot0{
				SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96),
				Tick:         -139,
				ProtocolFee:  500,
				LPFee:        3_000,
			},
		}, nil
	})
	if err != nil {
		t.Fatalf("newCLI() error = %v", err)
	}
	var output bytes.Buffer
	if err := applicationCLI.SetIO(strings.NewReader(""), &output, &bytes.Buffer{}); err != nil {
		t.Fatalf("SetIO() error = %v", err)
	}

	err = applicationCLI.RunArgs([]string{
		"v4", "get-slot0",
		"--http-url", "https://rpc.example",
		"--chain-id", "1",
		"--currency0", "0x0000000000000000000000000000000000000001",
		"--currency1", "0x0000000000000000000000000000000000000002",
		"--fee", "3000",
		"--tick-spacing", "60",
		"--block-number", "12345",
	})
	if err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}
	if received.HTTPURL != "https://rpc.example" || received.ChainID != 1 {
		t.Fatalf("received connection input = %+v", received)
	}
	if received.BlockNumber == nil || received.BlockNumber.Cmp(big.NewInt(12_345)) != 0 {
		t.Fatalf("received block number = %v, want 12345", received.BlockNumber)
	}
	if received.PoolKey.Fee != 3_000 || received.PoolKey.TickSpacing != 60 || received.PoolKey.Hooks != (common.Address{}) {
		t.Fatalf("received pool key = %+v", received.PoolKey)
	}
	for _, expected := range []string{"pool_id", "sqrt_price_x96", "79228162514264337593543950336", "tick", "-139", "protocol_fee", "500", "lp_fee", "3000"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output = %q, want substring %q", output.String(), expected)
		}
	}
}

func TestGetSlot0CommandUsesLatestBlockByDefault(t *testing.T) {
	t.Parallel()

	applicationCLI, err := newCLI(func(_ context.Context, input getSlot0Input) (getSlot0Output, error) {
		if input.BlockNumber != nil {
			t.Fatalf("BlockNumber = %v, want nil", input.BlockNumber)
		}
		poolID, err := input.PoolKey.ID()
		if err != nil {
			return getSlot0Output{}, err
		}
		return getSlot0Output{PoolID: poolID, Slot0: myUniswapV4.Slot0{SqrtPriceX96: big.NewInt(1)}}, nil
	})
	if err != nil {
		t.Fatalf("newCLI() error = %v", err)
	}
	var output bytes.Buffer
	if err := applicationCLI.SetIO(strings.NewReader(""), &output, &bytes.Buffer{}); err != nil {
		t.Fatalf("SetIO() error = %v", err)
	}

	if err := applicationCLI.RunArgs(validGetSlot0Args()); err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}
	if !strings.Contains(output.String(), "latest") {
		t.Fatalf("output = %q, want latest", output.String())
	}
}

func TestGetSlot0CommandRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{name: "missing HTTP URL", args: withoutOption(validGetSlot0Args(), "--http-url"), wantError: "option_http_url=empty"},
		{name: "invalid chain ID", args: replaceOption(validGetSlot0Args(), "--chain-id", "invalid"), wantError: "chain_id=invalid"},
		{name: "invalid fee", args: replaceOption(validGetSlot0Args(), "--fee", "invalid"), wantError: "fee=invalid"},
		{name: "invalid tick spacing", args: replaceOption(validGetSlot0Args(), "--tick-spacing", "0"), wantError: "tick_spacing=out_of_range"},
		{name: "invalid block number", args: append(validGetSlot0Args(), "--block-number=-1"), wantError: "block_number=invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			applicationCLI, err := newCLI(func(context.Context, getSlot0Input) (getSlot0Output, error) {
				t.Fatal("runner was called for invalid input")
				return getSlot0Output{}, nil
			})
			if err != nil {
				t.Fatalf("newCLI() error = %v", err)
			}
			if err := applicationCLI.SetIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
				t.Fatalf("SetIO() error = %v", err)
			}

			err = applicationCLI.RunArgs(test.args)
			if err == nil {
				t.Fatal("RunArgs() error = nil")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("RunArgs() error = %q, want substring %q", err, test.wantError)
			}
		})
	}
}

func validGetSlot0Args() []string {
	return []string{
		"v4", "get-slot0",
		"--http-url", "https://rpc.example",
		"--chain-id", "1",
		"--currency0", "0x0000000000000000000000000000000000000001",
		"--currency1", "0x0000000000000000000000000000000000000002",
		"--fee", "3000",
		"--tick-spacing", "60",
	}
}

func withoutOption(args []string, optionName string) []string {
	result := append([]string(nil), args...)
	for i := 0; i < len(result)-1; i++ {
		if result[i] == optionName {
			return append(result[:i], result[i+2:]...)
		}
	}
	return result
}

func replaceOption(args []string, optionName, value string) []string {
	result := append([]string(nil), args...)
	for i := 0; i < len(result)-1; i++ {
		if result[i] == optionName {
			result[i+1] = value
			return result
		}
	}
	return result
}
