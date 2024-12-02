package rpc

import (
	"context"
	"encoding/json"
	"fmt"
)

type PendingResult struct {
	request *inflightRequest
}

func newPendingResult(request *inflightRequest) *PendingResult {
	return &PendingResult{
		request: request,
	}
}

func (p *PendingResult) GetResultBlocking(result interface{}) error {
	select {
	case <-p.request.Done:
		if p.request.Error != nil {
			return p.request.Error
		} else {
			err := json.Unmarshal(p.request.Result, result)
			if err != nil {
				fmt.Sprintf("hi")
			}
			return err
		}
	case <-p.request.Context.Done():
		return p.request.Context.Err()
	}
}

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

type inflightRequest struct {
	// Done is used to signal to each interested worker that the request is completed (possibly with error).
	Done    chan struct{}
	Error   error
	Result  []byte
	Context context.Context
}
