package rpc

import (
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/context"
	"sync"
	"time"
)

const maxRetries = 3

/*
ClientPool is an Ethereum JSON-RPC provider that provides automatic connection pooling and request deduplication.
*/
type ClientPool struct {
	rpcClients       []*rpc.Client
	currentClientIdx int
	clientLock       sync.Mutex

	inflightRequests map[requestKey]*inflightRequest
	inflightLock     sync.Mutex

	endpoint   string
	maxRetries int
}

func NewClientPool(endpoint string, poolSize uint) (*ClientPool, error) {
	pool := &ClientPool{
		rpcClients:       make([]*rpc.Client, poolSize),
		clientLock:       sync.Mutex{},
		inflightRequests: make(map[requestKey]*inflightRequest),
		inflightLock:     sync.Mutex{},
		endpoint:         endpoint,
		maxRetries:       maxRetries,
	}

	// dial out
	for i := uint(0); i < poolSize; i++ {
		client, err := rpc.Dial(endpoint)
		if err != nil {
			return nil, err
		}
		pool.rpcClients[i] = client
	}

	return pool, nil
}

/*
ExecuteRequestBlocking makes a blocking RPC request and stores the result in the result interface pointer.
If there is an existing request on the wire with the same method/args, the calling thread will be blocked until it has
completed.
*/
func (c *ClientPool) ExecuteRequestBlocking(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	pending, err := c.ExecuteRequestAsync(ctx, method, args...)
	if err != nil {
		return err
	} else {
		return pending.GetResultBlocking(result)
	}
}

/*
ExecuteRequestAsync makes a non-blocking RPC request whose result may be obtained from interacting with *PendingResult.
If there is an existing request on the wire with the same method/args, this function will return a PendingResult linked
to that request.
*/
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
		client := c.getClient()

		go c.launchRequest(client, inflight, method, args...)
		return newPendingResult(inflight), nil
	}
}

// getClient obtains the next client in the round-robin of clients in the pool
func (c *ClientPool) getClient() *rpc.Client {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()

	client := c.rpcClients[c.currentClientIdx]
	c.currentClientIdx = (c.currentClientIdx + 1) % len(c.rpcClients)

	return client
}

// launchRequest performs the actual RPC request, storing the results of the request in the inflightRequest
func (c *ClientPool) launchRequest(
	client *rpc.Client,
	request *inflightRequest,
	method string,
	args ...interface{}) {
	defer close(request.Done)

	var err error
	var result string
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		err = client.CallContext(request.Context, &result, method, args...)
		if err == nil {
			request.Result = []byte("\"" + result + "\"")
			return
		}
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
	request.Error = err
}
