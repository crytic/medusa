package fork

import (
	"context"
	"github.com/crytic/medusa/chain/fork/rpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

type RemoteStateQuery interface {
	GetStorageAt(common.Address, common.Hash) (common.Hash, error)
	GetStateObject(common.Address) (*uint256.Int, uint64, []byte, error)
}

type RemoteStateRPCQuery struct {
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
	poolSize uint) (*RemoteStateRPCQuery, error) {
	clientPool, err := rpc.NewClientPool(url, poolSize)
	if err != nil {
		return nil, err
	}

	return &RemoteStateRPCQuery{
		context:    ctx,
		clientPool: clientPool,
		//height:           hexutil.Uint64(height).String(),
		height:           "latest",
		slotCache:        newSlotCache(),
		stateObjectCache: newStateObjectCache(),
	}, nil
}

func (q *RemoteStateRPCQuery) GetStorageAt(addr common.Address, slot common.Hash) (common.Hash, error) {
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

func (q *RemoteStateRPCQuery) GetStateObject(addr common.Address) (*uint256.Int, uint64, []byte, error) {
	obj, err := q.stateObjectCache.GetStateObject(addr)
	//addr = common.BytesToAddress(common.FromHex("0x4838b106fce9647bdf1e7877bf73ce8b0bad5f97"))
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
