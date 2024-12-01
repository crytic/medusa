package rpc

import (
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/context"
	"sync"
	"time"
)

const maxRetries = 3

type ClientPool struct {
	clients     []*ethclient.Client
	rpcClients  []*rpc.Client
	clientReady []bool
	lock        sync.Mutex

	inflightRequests map[requestKey]*inflightRequest
	inflightLock     sync.Mutex

	endpoint   string
	maxRetries int
}

func NewClientPool(endpoint string, poolSize int) (*ClientPool, error) {
	pool := &ClientPool{
		clients:          make([]*ethclient.Client, poolSize),
		rpcClients:       make([]*rpc.Client, poolSize),
		lock:             sync.Mutex{},
		inflightRequests: make(map[requestKey]*inflightRequest),
		inflightLock:     sync.Mutex{},
		endpoint:         endpoint,
		maxRetries:       maxRetries,
	}

	for i := 0; i < poolSize; i++ {
		rpcClient, err := rpc.Dial(endpoint)
		if err != nil {
			// todo: we may want to close the clients in this error case and generally
			return nil, fmt.Errorf("error when creating rpc client: %w", err)
		}

		client := ethclient.NewClient(rpcClient)
		pool.clients[i] = client
		pool.rpcClients[i] = rpcClient
		pool.clientReady[i] = true
	}
	return pool, nil
}

func (c *ClientPool) ExecuteRequestBlocking(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	pending, err := c.ExecuteRequestAsync(ctx, method, args)
	if err != nil {
		return err
	} else {
		return pending.GetResultBlocking(result)
	}
}

func (c *ClientPool) ExecuteRequestAsync(ctx context.Context, method string, args ...interface{}) (*PendingResult, error) {
	key, err := makeRequestKey(method, args...)
	if err != nil {
		return nil, err
	}

	// check for in-flight requests
	c.inflightLock.Lock()
	if inflight, exists := c.inflightRequests[key]; exists {
		c.inflightLock.Unlock()
		return newPendingResult(inflight), nil
	} else {
		// no inflight requests
		inflight = &inflightRequest{
			Done:    make(chan struct{}),
			Context: ctx,
		}
		c.inflightRequests[key] = inflight
		c.inflightLock.Unlock()

		go c.launchRequest(c.getClient(ctx), inflight, method, args)
		return newPendingResult(inflight), nil
	}
}

func (c *ClientPool) getClient(ctx context.Context) *rpc.Client {
	workerIndex := ctx.Value("workerIndex").(int)

	// defensive code
	if workerIndex >= len(c.clients) {
		panic("worker index out of range")
	}

	return c.rpcClients[workerIndex]
}

func (c *ClientPool) launchRequest(
	client *rpc.Client,
	request *inflightRequest,
	method string,
	args ...interface{}) {
	defer close(request.Done)

	var err error
	var result interface{}
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		err = client.CallContext(request.Context, result, method, args...)
		if err == nil {
			request.Result = result
			return
		}
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
	request.Error = err
}
