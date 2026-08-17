package v4

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	errTestSubscribeLogs = errors.New("test subscribe logs error")
	errTestSubscription  = errors.New("test subscription error")
)

type fakeSwapSourceSubscription struct {
	errs     chan error
	stopped  chan struct{}
	stopOnce sync.Once
}

func (s *fakeSwapSourceSubscription) Err() <-chan error {
	return s.errs
}

func (s *fakeSwapSourceSubscription) Unsubscribe() {
	s.stopOnce.Do(func() {
		close(s.stopped)
	})
}

type subscribeSwapWSRPCClient struct {
	query        ethereum.FilterQuery
	logs         chan<- types.Log
	subscription ethereum.Subscription
	err          error
}

func (c *subscribeSwapWSRPCClient) SubscribeFilterLogs(_ context.Context, query ethereum.FilterQuery, logs chan<- types.Log) (ethereum.Subscription, error) {
	c.query = query
	c.logs = logs
	return c.subscription, c.err
}

func TestSubscribeSwaps(t *testing.T) {
	t.Parallel()

	params := validClientParams()
	poolID, err := params.PoolManager.PoolKeys[0].ID()
	if err != nil {
		t.Fatalf("PoolKey.ID() error = %v", err)
	}
	source := newFakeSwapSourceSubscription()
	wsRPCClient := &subscribeSwapWSRPCClient{subscription: source}
	params.WSRPCClient = wsRPCClient
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	subscription, err := client.WS.SubscribeSwaps(context.Background())
	if err != nil {
		t.Fatalf("SubscribeSwaps() error = %v", err)
	}
	defer subscription.Unsubscribe()

	if len(wsRPCClient.query.Addresses) != 1 || wsRPCClient.query.Addresses[0] != params.PoolManager.Address {
		t.Fatalf("query addresses = %v, want [%s]", wsRPCClient.query.Addresses, params.PoolManager.Address.Hex())
	}
	if len(wsRPCClient.query.Topics) != 2 || wsRPCClient.query.Topics[0][0] != swapEventSignatureHash() || wsRPCClient.query.Topics[1][0] != poolID.Hash() {
		t.Fatalf("query topics = %v", wsRPCClient.query.Topics)
	}

	eventLog := newSwapLog(t)
	eventLog.Address = params.PoolManager.Address
	eventLog.Topics[1] = poolID.Hash()
	wsRPCClient.logs <- eventLog

	select {
	case swap := <-subscription.Swaps():
		if swap.PoolID != poolID {
			t.Fatalf("swap pool ID = %s, want %s", swap.PoolID.Hex(), poolID.Hex())
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Swap event")
	}
}

func TestSubscribeSwapsReportsInvalidLog(t *testing.T) {
	t.Parallel()

	params := validClientParams()
	source := newFakeSwapSourceSubscription()
	wsRPCClient := &subscribeSwapWSRPCClient{subscription: source}
	params.WSRPCClient = wsRPCClient
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	subscription, err := client.WS.SubscribeSwaps(context.Background())
	if err != nil {
		t.Fatalf("SubscribeSwaps() error = %v", err)
	}

	eventLog := newSwapLog(t)
	eventLog.Address = common.HexToAddress("0x01")
	wsRPCClient.logs <- eventLog

	select {
	case err := <-subscription.Err():
		if err == nil || !strings.Contains(err.Error(), "pool_manager=invalid") {
			t.Fatalf("subscription error = %v, want pool manager error", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for decode error")
	}
	waitForStop(t, source)
}

func TestSubscribeSwapsReportsSourceError(t *testing.T) {
	t.Parallel()

	params := validClientParams()
	source := newFakeSwapSourceSubscription()
	wsRPCClient := &subscribeSwapWSRPCClient{subscription: source}
	params.WSRPCClient = wsRPCClient
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	subscription, err := client.WS.SubscribeSwaps(context.Background())
	if err != nil {
		t.Fatalf("SubscribeSwaps() error = %v", err)
	}

	source.errs <- errTestSubscription
	select {
	case err := <-subscription.Err():
		if !errors.Is(err, errTestSubscription) {
			t.Fatalf("errors.Is() = false, want wrapped source error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for source error")
	}
	waitForStop(t, source)
}

func TestSubscribeSwapsStopsWithContext(t *testing.T) {
	t.Parallel()

	params := validClientParams()
	source := newFakeSwapSourceSubscription()
	params.WSRPCClient = &subscribeSwapWSRPCClient{subscription: source}
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	subscription, err := client.WS.SubscribeSwaps(ctx)
	if err != nil {
		t.Fatalf("SubscribeSwaps() error = %v", err)
	}

	cancel()
	waitForStop(t, source)
	select {
	case _, ok := <-subscription.Swaps():
		if ok {
			t.Fatal("Swaps() channel remains open")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Swap channel closure")
	}
}

func TestSubscribeSwapsWrapsCreationError(t *testing.T) {
	t.Parallel()

	params := validClientParams()
	params.WSRPCClient = &subscribeSwapWSRPCClient{err: errTestSubscribeLogs}
	client, err := NewClient(params)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.WS.SubscribeSwaps(context.Background())
	if err == nil {
		t.Fatal("SubscribeSwaps() error = nil")
	}
	if !errors.Is(err, errTestSubscribeLogs) {
		t.Fatalf("errors.Is() = false, want wrapped creation error: %v", err)
	}
}

func newFakeSwapSourceSubscription() *fakeSwapSourceSubscription {
	return &fakeSwapSourceSubscription{
		errs:    make(chan error, 1),
		stopped: make(chan struct{}),
	}
}

func waitForStop(t *testing.T, source *fakeSwapSourceSubscription) {
	t.Helper()

	select {
	case <-source.stopped:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for source unsubscribe")
	}
}
