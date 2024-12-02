package state

import (
	"context"
	"github.com/crytic/medusa/chain/state/rpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

type StateBackend interface {
	GetStorageAt(common.Address, common.Hash) (common.Hash, error)
	GetStateObject(common.Address) (*uint256.Int, uint64, []byte, error)
}

type RPCBackend struct {
	context    context.Context
	clientPool *rpc.ClientPool
	height     string

	slotCache        *slotCacheThreadSafe
	stateObjectCache *stateObjectCacheThreadSafe
}

func NewRemoteStateRPCQuery(
	ctx context.Context,
	url string,
	height uint64,
	poolSize uint) (*RPCBackend, error) {
	clientPool, err := rpc.NewClientPool(url, poolSize)
	if err != nil {
		return nil, err
	}

	return &RPCBackend{
		context:          ctx,
		clientPool:       clientPool,
		height:           hexutil.Uint64(height).String(),
		slotCache:        newSlotCache(),
		stateObjectCache: newStateObjectCache(),
	}, nil
}

func (q *RPCBackend) GetStorageAt(addr common.Address, slot common.Hash) (common.Hash, error) {
	data, err := q.slotCache.GetSlotData(addr, slot)
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
			q.slotCache.WriteSlotData(addr, slot, resultCast)
			return resultCast, nil
		}
	}
}

func (q *RPCBackend) GetStateObject(addr common.Address) (*uint256.Int, uint64, []byte, error) {
	obj, err := q.stateObjectCache.GetStateObject(addr)
	if err == nil {
		return obj.Balance, uint64(obj.Nonce), obj.Code, nil
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
		q.stateObjectCache.WriteStateObject(
			addr,
			remoteStateObject{
				balanceTyped,
				uint64(nonce),
				code,
			})
		return balanceTyped, uint64(nonce), code, nil
	}
}
