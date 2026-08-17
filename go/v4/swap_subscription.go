package v4

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const swapLogBufferSize = 64

type SwapSubscription struct {
	swaps        chan Swap
	errs         chan error
	cancel       context.CancelFunc
	subscription ethereum.Subscription
	stopOnce     sync.Once
}

// Swaps returns the decoded Swap event channel.
//
// Returns:
//   - Swap event channel, closed when the subscription stops.
//
// Version:
//   - 2026-08-17: Added.
func (s *SwapSubscription) Swaps() <-chan Swap {
	if s == nil {
		return nil
	}
	return s.swaps
}

// Err returns the terminal subscription error channel.
//
// Returns:
//   - Buffered error channel, closed when the subscription stops.
//
// Version:
//   - 2026-08-17: Added.
func (s *SwapSubscription) Err() <-chan error {
	if s == nil {
		return nil
	}
	return s.errs
}

// Unsubscribe stops the Swap subscription.
//
// Version:
//   - 2026-08-17: Added.
func (s *SwapSubscription) Unsubscribe() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.cancel()
		s.subscription.Unsubscribe()
	})
}

// SubscribeSwaps subscribes to Swap events for the configured Uniswap v4 pools.
//
// Parameters:
//   - ctx: subscription context; nil uses context.Background.
//
// Returns:
//   - Managed Swap subscription.
//   - Subscription creation error.
//
// Version:
//   - 2026-08-17: Added.
func (c *WSClient) SubscribeSwaps(ctx context.Context) (*SwapSubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to subscribe uniswap v4 swaps: client=null")
	}
	if c.rpc == nil {
		return nil, fmt.Errorf("failed to subscribe uniswap v4 swaps: ws_rpc_client=null")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	poolIDTopics, err := poolIDTopics(c.poolKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe uniswap v4 swaps: %w", err)
	}

	subscriptionCtx, cancel := context.WithCancel(ctx)
	logs := make(chan types.Log, swapLogBufferSize)
	source, err := c.rpc.SubscribeFilterLogs(subscriptionCtx, ethereum.FilterQuery{
		Addresses: []common.Address{c.poolManager},
		Topics: [][]common.Hash{
			{swapEventSignatureHash()},
			poolIDTopics,
		},
	}, logs)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to subscribe uniswap v4 swaps: %w", err)
	}
	if source == nil {
		cancel()
		return nil, fmt.Errorf("failed to subscribe uniswap v4 swaps: subscription=null")
	}

	subscription := &SwapSubscription{
		swaps:        make(chan Swap, swapLogBufferSize),
		errs:         make(chan error, 1),
		cancel:       cancel,
		subscription: source,
	}
	go c.consumeSwapLogs(subscriptionCtx, logs, subscription)

	return subscription, nil
}

func (c *WSClient) consumeSwapLogs(ctx context.Context, logs <-chan types.Log, subscription *SwapSubscription) {
	defer close(subscription.swaps)
	defer close(subscription.errs)
	defer subscription.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-subscription.subscription.Err():
			if ok && err != nil {
				subscription.errs <- fmt.Errorf("failed to consume uniswap v4 swap subscription: %w", err)
			}
			return
		case eventLog, ok := <-logs:
			if !ok {
				subscription.errs <- fmt.Errorf("failed to consume uniswap v4 swap subscription: logs_channel=closed")
				return
			}

			swap, err := c.decodeSubscribedSwap(eventLog)
			if err != nil {
				subscription.errs <- fmt.Errorf("failed to consume uniswap v4 swap subscription: %w", err)
				return
			}
			select {
			case subscription.swaps <- swap:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (c *WSClient) decodeSubscribedSwap(eventLog types.Log) (Swap, error) {
	if eventLog.Address != c.poolManager {
		return Swap{}, fmt.Errorf("failed to decode subscribed uniswap v4 swap: pool_manager=invalid")
	}

	swap, err := DecodeSwapLog(eventLog)
	if err != nil {
		return Swap{}, fmt.Errorf("failed to decode subscribed uniswap v4 swap: %w", err)
	}
	configured, err := hasPoolID(c.poolKeys, swap.PoolID)
	if err != nil {
		return Swap{}, fmt.Errorf("failed to decode subscribed uniswap v4 swap: %w", err)
	}
	if !configured {
		return Swap{}, fmt.Errorf("failed to decode subscribed uniswap v4 swap: pool_id=unconfigured")
	}

	return swap, nil
}
