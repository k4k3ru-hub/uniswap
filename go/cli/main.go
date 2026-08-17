package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	myCLI "github.com/k4k3ru-hub/cli/go"
	myOnchainEVM "github.com/k4k3ru-hub/onchain/go/evm"
	myUniswapV4 "github.com/k4k3ru-hub/uniswap/go/v4"
	"github.com/k4k3ru-hub/uniswap/go/v4/deployment"
	"github.com/k4k3ru-hub/uniswap/go/v4/protocol"
)

const (
	optionHTTPURL     = "http-url"
	optionChainID     = "chain-id"
	optionCurrency0   = "currency0"
	optionCurrency1   = "currency1"
	optionFee         = "fee"
	optionTickSpacing = "tick-spacing"
	optionHooks       = "hooks"
	optionBlockNumber = "block-number"
)

type getSlot0Input struct {
	HTTPURL     string
	ChainID     uint64
	PoolKey     protocol.PoolKey
	BlockNumber *big.Int
}

type getSlot0Output struct {
	PoolID protocol.PoolID
	Slot0  myUniswapV4.Slot0
}

type getSlot0Runner func(context.Context, getSlot0Input) (getSlot0Output, error)

func main() {
	applicationCLI, err := newCLI(executeGetSlot0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := applicationCLI.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCLI(runner getSlot0Runner) (*myCLI.CLI, error) {
	if runner == nil {
		return nil, fmt.Errorf("failed to create uniswap cli: get_slot0_runner=null")
	}

	applicationCLI := myCLI.NewCLI(nil)
	v4Command := myCLI.NewCommand("v4")
	v4Command.SetUsage("Interact with Uniswap v4.")
	getSlot0Command := myCLI.NewCommand("get-slot0")
	getSlot0Command.SetUsage("Get the current Slot0 state for a Uniswap v4 pool.")
	getSlot0Command.SetAction(newGetSlot0Action(runner))

	options := []struct {
		name   string
		option myCLI.Option
	}{
		{optionHTTPURL, myCLI.Option{Alias: "u", Description: "EVM HTTP RPC URL."}},
		{optionChainID, myCLI.Option{Alias: "n", Description: "EVM chain ID."}},
		{optionCurrency0, myCLI.Option{Description: "Pool currency0 address."}},
		{optionCurrency1, myCLI.Option{Description: "Pool currency1 address."}},
		{optionFee, myCLI.Option{Alias: "f", Description: "Pool fee or dynamic fee flag."}},
		{optionTickSpacing, myCLI.Option{Alias: "t", Description: "Pool tick spacing."}},
		{optionHooks, myCLI.Option{DefaultValue: "0x0000000000000000000000000000000000000000", Description: "Pool hooks address."}},
		{optionBlockNumber, myCLI.Option{Alias: "b", Description: "Optional block number; latest when omitted."}},
	}
	for _, item := range options {
		if err := getSlot0Command.AddOption(item.name, item.option); err != nil {
			return nil, fmt.Errorf("failed to create uniswap cli: %w", err)
		}
	}
	if err := getSlot0Command.SetArgumentCount(0, 0); err != nil {
		return nil, fmt.Errorf("failed to create uniswap cli: %w", err)
	}
	if err := v4Command.AddCommand(getSlot0Command); err != nil {
		return nil, fmt.Errorf("failed to create uniswap cli: %w", err)
	}
	if err := applicationCLI.Root().AddCommand(v4Command); err != nil {
		return nil, fmt.Errorf("failed to create uniswap cli: %w", err)
	}

	return applicationCLI, nil
}

func newGetSlot0Action(runner getSlot0Runner) myCLI.CommandFunc {
	return func(cliContext *myCLI.Context) error {
		input, err := parseGetSlot0Input(cliContext)
		if err != nil {
			return fmt.Errorf("failed to execute get-slot0 command: %w", err)
		}

		output, err := runner(context.Background(), input)
		if err != nil {
			return fmt.Errorf("failed to execute get-slot0 command: %w", err)
		}

		blockNumber := "latest"
		if input.BlockNumber != nil {
			blockNumber = input.BlockNumber.String()
		}
		if err := myCLI.OutputTableTo(cliContext.Output(), []string{"FIELD", "VALUE"}, [][]any{
			{"chain_id", input.ChainID},
			{"block_number", blockNumber},
			{"pool_id", output.PoolID.Hex()},
			{"sqrt_price_x96", output.Slot0.SqrtPriceX96.String()},
			{"tick", output.Slot0.Tick},
			{"protocol_fee", output.Slot0.ProtocolFee},
			{"lp_fee", output.Slot0.LPFee},
		}); err != nil {
			return fmt.Errorf("failed to execute get-slot0 command: %w", err)
		}

		return nil
	}
}

func parseGetSlot0Input(cliContext *myCLI.Context) (getSlot0Input, error) {
	if cliContext == nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: cli_context=null")
	}

	httpURL, err := requiredOption(cliContext, optionHTTPURL)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w", err)
	}
	chainIDValue, err := requiredOption(cliContext, optionChainID)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w", err)
	}
	chainID, err := strconv.ParseUint(chainIDValue, 10, 64)
	if err != nil || chainID == 0 {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: chain_id=invalid")
	}
	currency0Value, err := requiredOption(cliContext, optionCurrency0)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w", err)
	}
	currency0, err := protocol.ParseCurrency(currency0Value)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w: currency=currency0", err)
	}
	currency1Value, err := requiredOption(cliContext, optionCurrency1)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w", err)
	}
	currency1, err := protocol.ParseCurrency(currency1Value)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w: currency=currency1", err)
	}
	feeValue, err := requiredOption(cliContext, optionFee)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w", err)
	}
	fee, err := strconv.ParseUint(feeValue, 10, 32)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: fee=invalid")
	}
	tickSpacingValue, err := requiredOption(cliContext, optionTickSpacing)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w", err)
	}
	tickSpacing, err := strconv.ParseInt(tickSpacingValue, 10, 32)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: tick_spacing=invalid")
	}
	hooksValue, err := requiredOption(cliContext, optionHooks)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w", err)
	}
	hooks, err := myOnchainEVM.ParseAddress(hooksValue)
	if err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w: address=hooks", err)
	}

	poolKey := protocol.PoolKey{
		Currency0:   currency0,
		Currency1:   currency1,
		Fee:         uint32(fee),
		TickSpacing: int32(tickSpacing),
		Hooks:       hooks,
	}
	if err := poolKey.Validate(); err != nil {
		return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: %w", err)
	}

	var blockNumber *big.Int
	if blockNumberOption, ok := cliContext.Option(optionBlockNumber); ok && blockNumberOption.IsSet {
		blockNumber = new(big.Int)
		if _, ok := blockNumber.SetString(strings.TrimSpace(blockNumberOption.Value), 10); !ok || blockNumber.Sign() < 0 {
			return getSlot0Input{}, fmt.Errorf("failed to parse get-slot0 input: block_number=invalid")
		}
	}

	return getSlot0Input{
		HTTPURL:     httpURL,
		ChainID:     chainID,
		PoolKey:     poolKey,
		BlockNumber: blockNumber,
	}, nil
}

func requiredOption(cliContext *myCLI.Context, name string) (string, error) {
	option, ok := cliContext.Option(name)
	if !ok || strings.TrimSpace(option.Value) == "" {
		return "", fmt.Errorf("failed to get required command option: option_%s=empty", strings.ReplaceAll(name, "-", "_"))
	}
	return strings.TrimSpace(option.Value), nil
}

func executeGetSlot0(ctx context.Context, input getSlot0Input) (getSlot0Output, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	officialDeployment, err := deployment.ByChainID(input.ChainID)
	if err != nil {
		return getSlot0Output{}, fmt.Errorf("failed to get slot0 from uniswap v4: %w", err)
	}
	httpRPCClient, err := myOnchainEVM.NewHTTPClient(ctx, myOnchainEVM.HTTPConfig{URL: input.HTTPURL})
	if err != nil {
		return getSlot0Output{}, fmt.Errorf("failed to get slot0 from uniswap v4: %w", err)
	}
	httpClient, err := myUniswapV4.NewHTTPClient(myUniswapV4.HTTPClientParams{
		RPC: httpRPCClient,
		PoolManager: myUniswapV4.PoolManagerConfig{
			Address:   officialDeployment.PoolManager,
			StateView: officialDeployment.StateView,
			PoolKeys:  []protocol.PoolKey{input.PoolKey},
		},
	})
	if err != nil {
		return getSlot0Output{}, fmt.Errorf("failed to get slot0 from uniswap v4: %w", err)
	}
	poolID, err := input.PoolKey.ID()
	if err != nil {
		return getSlot0Output{}, fmt.Errorf("failed to get slot0 from uniswap v4: %w", err)
	}
	slot0, err := httpClient.GetSlot0(ctx, poolID, input.BlockNumber)
	if err != nil {
		return getSlot0Output{}, fmt.Errorf("failed to get slot0 from uniswap v4: %w", err)
	}

	return getSlot0Output{PoolID: poolID, Slot0: slot0}, nil
}
