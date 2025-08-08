package rpc

import (
	"context"
	"encoding/json"
)

/*
PendingResult defines an object that can be returned when calling the RPC asynchronously. It's kind of like a promise as
seen in other languages.
*/
type PendingResult struct {
	request *inflightRequest
}

func newPendingResult(request *inflightRequest) *PendingResult {
	return &PendingResult{
		request: request,
	}
}

/*
GetResultBlocking obtains the result from the client, blocking until the result or an error is available. Callers must
pass a pointer to their data through result. Note that if the fuzzer is shutting down, an error may be returned to
signify the context has been cancelled.
*/
func (p *PendingResult) GetResultBlocking(result interface{}) error {
	select {
	case <-p.request.Done:
		if p.request.Error != nil {
			return p.request.Error
		} else {
			err := json.Unmarshal(p.request.Result, result)
			return err
		}
	case <-p.request.Context.Done():
		return p.request.Context.Err()
	}
}

// requestKey defines a struct that can uniquely identify an Ethereum RPC request for request deduplication purposes.
type requestKey struct {
	Method string
	Args   string
}

func makeRequestKey(method string, args ...interface{}) (requestKey, error) {
	serialized, err := json.Marshal(args)
	if err != nil {
		return requestKey{}, err
	} else {
		return requestKey{Method: method, Args: string(serialized)}, nil
	}

}

// inflightRequest represents an HTTP-JSON request that is currently traversing the network.
type inflightRequest struct {
	// Done is used to signal to each interested worker that the request is completed (possibly with error).
	Done    chan struct{}
	Error   error
	Result  []byte
	Context context.Context
}
