package v4

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// FilterSwaps gets historical Swap events for the configured Uniswap v4 pools.
//
// Parameters:
//   - ctx: request context; nil uses context.Background in the injected HTTP RPC client.
//   - fromBlock: first block to query; nil uses the RPC default.
//   - toBlock: last block to query; nil uses the latest block.
//
// Returns:
//   - Decoded Swap events in RPC response order.
//   - Filter or decode error.
//
// Version:
//   - 2026-08-17: Added.
func (c *Client) FilterSwaps(ctx context.Context, fromBlock, toBlock *big.Int) ([]Swap, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to filter uniswap v4 swaps: client=null")
	}
	if c.httpRPCClient == nil {
		return nil, fmt.Errorf("failed to filter uniswap v4 swaps: http_rpc_client=null")
	}

	poolIDTopics, err := c.poolIDTopics()
	if err != nil {
		return nil, fmt.Errorf("failed to filter uniswap v4 swaps: %w", err)
	}

	logs, err := c.httpRPCClient.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: []common.Address{c.poolManager},
		Topics: [][]common.Hash{
			{swapEventSignatureHash()},
			poolIDTopics,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to filter uniswap v4 swaps: %w", err)
	}

	swaps := make([]Swap, 0, len(logs))
	for i, eventLog := range logs {
		if eventLog.Address != c.poolManager {
			return nil, fmt.Errorf("failed to filter uniswap v4 swaps: pool_manager=invalid: result_index=%d", i)
		}

		swap, err := DecodeSwapLog(eventLog)
		if err != nil {
			return nil, fmt.Errorf("failed to filter uniswap v4 swaps: %w: result_index=%d", err, i)
		}
		configured, err := c.hasPoolID(swap.PoolID)
		if err != nil {
			return nil, fmt.Errorf("failed to filter uniswap v4 swaps: %w: result_index=%d", err, i)
		}
		if !configured {
			return nil, fmt.Errorf("failed to filter uniswap v4 swaps: pool_id=unconfigured: result_index=%d", i)
		}

		swaps = append(swaps, swap)
	}

	return swaps, nil
}

func (c *Client) poolIDTopics() ([]common.Hash, error) {
	topics := make([]common.Hash, 0, len(c.poolKeys))
	for i, poolKey := range c.poolKeys {
		poolID, err := poolKey.ID()
		if err != nil {
			return nil, fmt.Errorf("failed to build uniswap v4 pool id topics: %w: pool_index=%d", err, i)
		}
		topics = append(topics, poolID.Hash())
	}

	return topics, nil
}
