package chain

import (
	"github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/events"
)

// TestChainEvents defines event emitters for a TestChain.
type TestChainEvents struct {
	// PendingBlockCreated emits events indicating a pending block was created on the chain.
	PendingBlockCreated events.EventEmitter[PendingBlockCreatedEvent]

	// PendingBlockAddedTx emits events indicating a pending block had a transaction added to it.
	PendingBlockAddedTx events.EventEmitter[PendingBlockAddedTxEvent]

	// PendingBlockCommitted emits events indicating a pending block was committed to the chain.
	PendingBlockCommitted events.EventEmitter[PendingBlockCommittedEvent]

	// PendingBlockDiscarded emits events indicating a pending block was discarded before being committed to the chain.
	PendingBlockDiscarded events.EventEmitter[PendingBlockDiscardedEvent]

	// BlocksRemoved emits events indicating a block(s) was removed from the chain.
	BlocksRemoved events.EventEmitter[BlocksRemovedEvent]

	// ContractDeploymentAddedEventEmitter emits events indicating a new contract was created on chain. This is called
	// alongside ContractDeploymentRemovedEventEmitter when contract deployment changes are detected. e.g. If a
	// contract is deployed and immediately destroyed within the same transaction, a ContractDeploymentsAddedEvent
	// will be fired, followed immediately by a ContractDeploymentsRemovedEvent.
	ContractDeploymentAddedEventEmitter events.EventEmitter[ContractDeploymentsAddedEvent]

	// ContractDeploymentAddedEventEmitter emits events indicating a previously deployed contract was removed
	// from the chain. This is called alongside ContractDeploymentAddedEventEmitter when contract deployment changes
	// are detected. e.g. If a contract is deployed and immediately destroyed within the same transaction, a
	// ContractDeploymentsAddedEvent will be fired, followed immediately by a ContractDeploymentsRemovedEvent.
	ContractDeploymentRemovedEventEmitter events.EventEmitter[ContractDeploymentsRemovedEvent]
}

// PendingBlockCreatedEvent describes an event where a new pending block was created, prior to any transactions being
// added to it.
type PendingBlockCreatedEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Block refers to the Chain's pending block which was just created.
	Block *types.Block
}

// PendingBlockAddedTxEvent describes an event where a pending block had a transaction added to it.
type PendingBlockAddedTxEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Block refers to the pending block which the transaction was added to.
	Block *types.Block

	// TransactionIndex describes the index of the transaction in the pending block.
	TransactionIndex int
}

// PendingBlockCommittedEvent describes an event where a pending block is committed to the chain as the new head.
type PendingBlockCommittedEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Block refers to the pending block that was committed to the Chain as the new head.
	Block *types.Block
}

// PendingBlockDiscardedEvent describes an event where a pending block is discarded from the chain before being committed.
type PendingBlockDiscardedEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Block refers to the pending block that was discarded before being committed to the Chain.
	Block *types.Block
}

// BlocksRemovedEvent describes an event where a block(s) is removed from the TestChain. This only considers internally
// committed blocks, not ones spoofed in between block number jumps.
type BlocksRemovedEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Blocks refers to the block(s) that was removed from the Chain.
	Blocks []*types.Block
}

// ContractDeploymentsAddedEvent describes an event where a contract has become available on the TestChain, either
// due to contract creation, or a self-destruct operation being reverted.
type ContractDeploymentsAddedEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Contract defines information for the contract which was deployed to the Chain.
	Contract *types.DeployedContractBytecode

	// DynamicDeployment describes whether this contract deployment was dynamic (e.g. `c = new MyContract()`) or was
	// because of a traditional transaction
	DynamicDeployment bool
}

// ContractDeploymentsRemovedEvent describes an event where a contract has become unavailable on the TestChain, either
// due to the reverting of a contract creation, or a self-destruct operation.
type ContractDeploymentsRemovedEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Contract defines information for the contract which was deployed to the Chain.
	Contract *types.DeployedContractBytecode
}
