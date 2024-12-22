package state

import (
	"context"
	"github.com/crytic/medusa/chain/state/object"
	"github.com/crytic/medusa/chain/state/rpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

/*
stateBackend defines an interface for fetching arbitrary state from a different source such as a remote RPC server or
K/V store.
*/
type stateBackend interface {
	GetStorageAt(common.Address, common.Hash) (common.Hash, error)
	GetStateObject(common.Address) (*uint256.Int, uint64, []byte, error)
}

var _ stateBackend = (*EmptyBackend)(nil)
var _ stateBackend = (*RPCBackend)(nil)

/*
RPCBackend defines a stateBackend for fetching state from a remote RPC server. It is locked to a single block height,
and caches data in-memory with no expiry.
*/
type RPCBackend struct {
	context    context.Context
	clientPool *rpc.ClientPool
	height     string

	cache object.StateCache
}

func NewRPCBackend(
	ctx context.Context,
	url string,
	height uint64,
	poolSize uint) (*RPCBackend, error) {
	clientPool, err := rpc.NewClientPool(url, poolSize)
	if err != nil {
		return nil, err
	}

	cache, err := object.NewPersistentCache(ctx, url, height)
	if err != nil {
		return nil, err
	}

	return &RPCBackend{
		context:    ctx,
		clientPool: clientPool,
		height:     hexutil.Uint64(height).String(),
		cache:      cache,
	}, nil
}

// newRPCBackendNoPersistence creates a new RPC backend that will not persist its cache to disk. used for tests.
// nolint:unused
func newRPCBackendNoPersistence(
	ctx context.Context,
	url string,
	height uint64,
	poolSize uint) (*RPCBackend, error) {
	clientPool, err := rpc.NewClientPool(url, poolSize)
	if err != nil {
		return nil, err
	}

	cache, err := object.NewNonPersistentCache()
	if err != nil {
		return nil, err
	}

	return &RPCBackend{
		context:    ctx,
		clientPool: clientPool,
		height:     hexutil.Uint64(height).String(),
		cache:      cache,
	}, nil
}

/*
GetStorageAt returns data stored in the remote RPC for the given address/slot.
Note that Ethereum RPC will return zero for slots that have never been written to or are associated with undeployed
contracts.
Errors may be network errors or a context cancelled error when the fuzzer is shutting down.
*/
func (q *RPCBackend) GetStorageAt(addr common.Address, slot common.Hash) (common.Hash, error) {
	data, err := q.cache.GetSlotData(addr, slot)
	if err == nil {
		return data, nil
	} else {
		method := "eth_getStorageAt"
		var result hexutil.Bytes
		err = q.clientPool.ExecuteRequestBlocking(q.context, &result, method, addr, slot, q.height)
		if err != nil {
			return common.Hash{}, err
		} else {
			resultCast := common.HexToHash(common.Bytes2Hex(result))
			err = q.cache.WriteSlotData(addr, slot, resultCast)
			return resultCast, err
		}
	}
}

/*
GetStateObject returns the data stored in the remote RPC for the specified state object
Note that the Ethereum RPC will return zero for accounts that do not exist.
Errors may be network errors or a context cancelled error when the fuzzer is shutting down.
*/
func (q *RPCBackend) GetStateObject(addr common.Address) (*uint256.Int, uint64, []byte, error) {
	obj, err := q.cache.GetStateObject(addr)
	if err == nil {
		return obj.Balance, obj.Nonce, obj.Code, nil
	} else {
		balance := hexutil.Big{}
		nonce := hexutil.Uint(0)
		code := hexutil.Bytes{}

		pendingBalance, err := q.clientPool.ExecuteRequestAsync(
			q.context,
			"eth_getBalance",
			addr,
			q.height)
		if err != nil {
			return nil, 0, nil, err
		}
		pendingNonce, err := q.clientPool.ExecuteRequestAsync(
			q.context,
			"eth_getTransactionCount",
			addr,
			q.height)
		if err != nil {
			return nil, 0, nil, err
		}

		pendingCode, err := q.clientPool.ExecuteRequestAsync(
			q.context,
			"eth_getCode",
			addr,
			q.height)
		if err != nil {
			return nil, 0, nil, err
		}

		err = pendingBalance.GetResultBlocking(&balance)
		if err != nil {
			return nil, 0, nil, err
		}
		balanceTyped := &uint256.Int{}
		balanceTyped.SetFromBig(balance.ToInt())

		err = pendingNonce.GetResultBlocking(&nonce)
		if err != nil {
			return nil, 0, nil, err
		}

		err = pendingCode.GetResultBlocking(&code)
		if err != nil {
			return nil, 0, nil, err
		}
		err = q.cache.WriteStateObject(
			addr,
			object.StateObject{
				Balance: balanceTyped,
				Nonce:   uint64(nonce),
				Code:    code,
			})
		return balanceTyped, uint64(nonce), code, err
	}
}
