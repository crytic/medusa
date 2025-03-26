package state

import (
	"fmt"

	"github.com/crytic/medusa-geth/common"
	gethState "github.com/crytic/medusa-geth/core/state"
	"github.com/holiman/uint256"
)

var _ gethState.RemoteStateProvider = (*RemoteStateProvider)(nil)

/*
RemoteStateProvider implements an import mechanism for state that was not written by a locally executed transaction.
This allows us to use the state of a remote RPC server for fork mode, or the state of some other serialized database.
It is consumed by medusa-geth's ForkStateDb.
This provider is snapshot-aware and will refuse to fetch certain data if it has reason to believe the local statedb
has newer data.
*/
type RemoteStateProvider struct {
	// stateBackend is used to fetch state when RemoteStateProvider believes the remote source to be canonical
	stateBackend stateBackend

	// stateObjBySnapshot keeps track of imported state objects by snapshot, thus allowing state objects to be
	// re-imported when their snapshot is reverted
	stateObjBySnapshot  map[int][]common.Address
	stateSlotBySnapshot map[int]map[common.Address][]common.Hash

	// stateObjsImported keeps track of all the state objects RemoteStateProvider has imported
	stateObjsImported map[common.Address]struct{}
	// stateSlotsImported keeps track of all the storage slots RemoteStateProvider has imported
	stateSlotsImported map[common.Address]map[common.Hash]struct{}

	// contractsDeployed keeps track of contracts that were deployed locally.
	contractsDeployed           map[common.Address]struct{}
	contractsDeployedBySnapshot map[int][]common.Address
}

func newRemoteStateProvider(stateBackend stateBackend) *RemoteStateProvider {
	return &RemoteStateProvider{
		stateBackend:                stateBackend,
		stateObjBySnapshot:          make(map[int][]common.Address),
		stateSlotBySnapshot:         make(map[int]map[common.Address][]common.Hash),
		stateObjsImported:           make(map[common.Address]struct{}),
		stateSlotsImported:          make(map[common.Address]map[common.Hash]struct{}),
		contractsDeployed:           make(map[common.Address]struct{}),
		contractsDeployedBySnapshot: make(map[int][]common.Address),
	}
}

/*
ImportStateObject attempts to import a state object from the backend. If the state object has already been imported and
its snapshot has not been reverted, this function will return an error with CannotQueryDirtyAccount set to true.
*/
func (s *RemoteStateProvider) ImportStateObject(
	addr common.Address,
	snapId int,
) (bal *uint256.Int, nonce uint64, code []byte, e *gethState.RemoteStateError) {
	if _, ok := s.stateObjsImported[addr]; ok {
		return nil, 0, nil, &gethState.RemoteStateError{
			CannotQueryDirtyAccount: true,
			Error: fmt.Errorf("state object %s was already imported",
				addr.Hex()),
		}
	}

	bal, nonce, code, err := s.stateBackend.GetStateObject(addr)
	if err == nil {
		s.recordDirtyStateObject(addr, snapId)
		return bal, nonce, code, nil
	} else {
		return uint256.NewInt(0), 0, nil, &gethState.RemoteStateError{
			CannotQueryDirtyAccount: false,
			Error:                   err,
		}
	}
}

/*
ImportStorageAt attempts to import a storage slot from the backend. If the slot has already been imported and its
snapshot has not been reverted, this function will return an error with CannotQueryDirtySlot set to true.
If the storage slot is associated with a contract that was deployed locally, this function will return an error with
CannotQueryDirtySlot set to true, since the remote database will never contain canonical slot data for a locally
deployed contract.
*/
func (s *RemoteStateProvider) ImportStorageAt(
	addr common.Address,
	slot common.Hash,
	snapId int,
) (common.Hash, *gethState.RemoteStorageError) {
	// if the contract was deployed locally, the RPC will not have data for its slots
	if _, exists := s.contractsDeployed[addr]; exists {
		return common.Hash{}, &gethState.RemoteStorageError{
			CannotQueryDirtySlot: true,
			Error: fmt.Errorf(
				"state slot %s of address %s cannot be remote-queried because the contract was deployed locally",
				slot.Hex(),
				addr.Hex(),
			),
		}
	}

	imported := s.isStateSlotImported(addr, slot)
	if imported {
		return common.Hash{}, &gethState.RemoteStorageError{
			CannotQueryDirtySlot: true,
			Error: fmt.Errorf(
				"state slot %s of address %s was already imported in snapshot %d",
				slot.Hex(),
				addr.Hex(),
				snapId,
			),
		}
	}
	data, err := s.stateBackend.GetStorageAt(addr, slot)
	if err == nil {
		s.recordDirtyStateSlot(addr, slot, snapId)
		return data, nil
	} else {
		return common.Hash{}, &gethState.RemoteStorageError{
			CannotQueryDirtySlot: false,
			Error:                err,
		}
	}
}

/*
MarkSlotWritten is used to notify the provider that a local transaction has written a value to the specified slot.
As long as the snapshot indicated by snapId is not reverted, the provider will now return "dirty" if ImportStorageAt is
called for the slot in the future.
*/
func (s *RemoteStateProvider) MarkSlotWritten(addr common.Address, slot common.Hash, snapId int) {
	s.recordDirtyStateSlot(addr, slot, snapId)
}

/*
MarkContractDeployed is used to notify the provider that a contract was locally deployed to the specified address.
As long as the snapshot indicated by snapId is not reverted, the provider will not return "dirty" if ImportStorageAt is
called for any slots associated with the contract.
*/
func (s *RemoteStateProvider) MarkContractDeployed(addr common.Address, snapId int) {
	s.recordContractDeployed(addr, snapId)
}

/*
NotifyRevertedToSnapshot is used to notify the provider that the state has been reverted back to snapId. The provider
uses this information to clear its import history up to and not including the provided snapId.
*/
func (s *RemoteStateProvider) NotifyRevertedToSnapshot(snapId int) {
	// purge all records down to and not including the provided snapId

	/* accounts */
	accountsToClear := make([]common.Address, 0)
	for sId, accounts := range s.stateObjBySnapshot {
		if sId > snapId {
			accountsToClear = append(accountsToClear, accounts...)
			delete(s.stateObjBySnapshot, sId)
		}
	}
	for _, addr := range accountsToClear {
		delete(s.stateObjsImported, addr)
	}

	/* state slots */
	accountSlotsToClear := make(map[common.Address][]common.Hash)
	for sId, accounts := range s.stateSlotBySnapshot {
		if sId > snapId {
			for addr, slots := range accounts {
				if _, ok := accountSlotsToClear[addr]; !ok {
					accountSlotsToClear[addr] = make([]common.Hash, 0, len(slots))
				}
				accountSlotsToClear[addr] = append(accountSlotsToClear[addr], slots...)
			}
			delete(s.stateSlotBySnapshot, sId)
		}
	}

	for addr, slots := range accountSlotsToClear {
		for _, slot := range slots {
			delete(s.stateSlotsImported[addr], slot)
		}
	}

	/* contract deploys */
	contractsToClear := make([]common.Address, 0)
	for sId, contracts := range s.contractsDeployedBySnapshot {
		if sId > snapId {
			contractsToClear = append(contractsToClear, contracts...)
			delete(s.contractsDeployedBySnapshot, sId)
		}
	}
	for _, contract := range contractsToClear {
		delete(s.contractsDeployed, contract)
	}
}

func (s *RemoteStateProvider) isStateSlotImported(addr common.Address, slot common.Hash) bool {
	if _, ok := s.stateSlotsImported[addr]; !ok {
		return false
	} else {
		if _, ok := s.stateSlotsImported[addr][slot]; !ok {
			return false
		} else {
			return true
		}
	}
}

func (s *RemoteStateProvider) recordDirtyStateObject(addr common.Address, snapId int) {
	s.stateObjsImported[addr] = struct{}{}
	if _, ok := s.stateObjBySnapshot[snapId]; !ok {
		s.stateObjBySnapshot[snapId] = make([]common.Address, 0)
	}
	s.stateObjBySnapshot[snapId] = append(s.stateObjBySnapshot[snapId], addr)
}

func (s *RemoteStateProvider) recordDirtyStateSlot(addr common.Address, slot common.Hash, snapId int) {
	if _, ok := s.stateSlotsImported[addr]; !ok {
		s.stateSlotsImported[addr] = make(map[common.Hash]struct{})
	}
	// If this slot has already been marked dirty, we don't want to do it again or overwrite the old snapId.
	// We only want to track the oldest snapId that made a slot dirty, since that is the only snapId that should
	// change the RemoteStateProvider's import behavior if reverted.
	if _, exists := s.stateSlotsImported[addr][slot]; exists {
		return
	}

	s.stateSlotsImported[addr][slot] = struct{}{}
	if _, ok := s.stateSlotBySnapshot[snapId]; !ok {
		s.stateSlotBySnapshot[snapId] = make(map[common.Address][]common.Hash)
	}
	if _, ok := s.stateSlotBySnapshot[snapId][addr]; !ok {
		s.stateSlotBySnapshot[snapId][addr] = make([]common.Hash, 0)
	}

	s.stateSlotBySnapshot[snapId][addr] = append(s.stateSlotBySnapshot[snapId][addr], slot)
}

func (s *RemoteStateProvider) recordContractDeployed(addr common.Address, snapId int) {
	s.contractsDeployed[addr] = struct{}{}
	if _, ok := s.contractsDeployedBySnapshot[snapId]; !ok {
		s.contractsDeployedBySnapshot[snapId] = make([]common.Address, 0)
	}
	s.contractsDeployedBySnapshot[snapId] = append(s.contractsDeployedBySnapshot[snapId], addr)
}
