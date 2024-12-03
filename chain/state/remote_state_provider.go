package state

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/holiman/uint256"
)

var _ state.RemoteStateProvider = (*RemoteStateProvider)(nil)

type RemoteStateProvider struct {
	stateBackend StateBackend

	stateObjBySnapshot  map[int][]common.Address
	stateSlotBySnapshot map[int]map[common.Address][]common.Hash

	stateObjsImported  map[common.Address]struct{}
	stateSlotsImported map[common.Address]map[common.Hash]struct{}

	contractsDeployed           map[common.Address]struct{}
	contractsDeployedBySnapshot map[int][]common.Address
}

func newRemoteStateProvider(stateBackend StateBackend) *RemoteStateProvider {
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

func (s *RemoteStateProvider) ImportStateObject(addr common.Address, snapId int) (bal *uint256.Int, nonce uint64, code []byte, e *state.RemoteStateError) {
	if existingSnap, ok := s.stateObjsImported[addr]; ok {
		return nil, 0, nil, &state.RemoteStateError{
			CannotQueryDirtyAccount: true,
			Error:                   fmt.Errorf("state object %s was already imported in snapshot %d", addr.Hex(), existingSnap),
		}
	}

	bal, nonce, code, err := s.stateBackend.GetStateObject(addr)
	if err == nil {
		s.recordImportedStateObject(addr, snapId)
		return bal, nonce, code, nil
	} else {
		return uint256.NewInt(0), 0, nil, &state.RemoteStateError{
			CannotQueryDirtyAccount: false,
			Error:                   err,
		}
	}
}

func (s *RemoteStateProvider) ImportStorageAt(addr common.Address, slot common.Hash, snapId int) (common.Hash, *state.RemoteStorageError) {
	// if the contract was deployed locally, the RPC will not have data for its slots
	if _, exists := s.contractsDeployed[addr]; exists {
		return common.Hash{}, &state.RemoteStorageError{
			CannotQueryDirtySlot: true,
			Error:                fmt.Errorf("state slot %s of address %s cannot be remote-queried because the contract was deployed locally", slot.Hex(), addr.Hex()),
		}
	}

	imported := s.isStateSlotImported(addr, slot)
	if imported {
		return common.Hash{}, &state.RemoteStorageError{
			CannotQueryDirtySlot: true,
			Error:                fmt.Errorf("state slot %s of address %s was already imported in snapshot %d", slot.Hex(), addr.Hex(), snapId),
		}
	}
	data, err := s.stateBackend.GetStorageAt(addr, slot)
	if err == nil {
		s.recordImportedStateSlot(addr, slot, snapId)
		return data, nil
	} else {
		return common.Hash{}, &state.RemoteStorageError{
			CannotQueryDirtySlot: false,
			Error:                err,
		}
	}
}

func (s *RemoteStateProvider) MarkSlotWritten(addr common.Address, slot common.Hash, snapId int) {
	s.recordImportedStateSlot(addr, slot, snapId)
}

func (s *RemoteStateProvider) MarkContractDeployed(addr common.Address, snapId int) {
	s.recordContractDeployed(addr, snapId)
}

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

func (s *RemoteStateProvider) recordImportedStateObject(addr common.Address, snapId int) {
	s.stateObjsImported[addr] = struct{}{}
	if _, ok := s.stateObjBySnapshot[snapId]; !ok {
		s.stateObjBySnapshot[snapId] = make([]common.Address, 0)
	}
	s.stateObjBySnapshot[snapId] = append(s.stateObjBySnapshot[snapId], addr)
}

func (s *RemoteStateProvider) recordImportedStateSlot(addr common.Address, slot common.Hash, snapId int) {
	if _, ok := s.stateSlotsImported[addr]; !ok {
		s.stateSlotsImported[addr] = make(map[common.Hash]struct{})
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
