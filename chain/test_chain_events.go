package chain

import (
	"github.com/trailofbits/medusa/chain/types"
)

// BlockMiningEvent describes an event where a new block has been requested to be mined in the TestChain. This only
// considers committed blocks.
type BlockMiningEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain
}

// BlockMinedEvent describes an event where a new block is added to the TestChain. This only considers committed blocks.
type BlockMinedEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Block refers to the block that was added to the Chain.
	Block *types.Block
}

// BlocksRemovedEvent describes an event where a block is removed from the TestChain. This only considers internally
// committed blocks, not ones spoofed in between block number jumps.
type BlocksRemovedEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Blocks refers to the block that was removed from the Chain.
	Blocks []*types.Block
}

// ContractDeploymentsAddedEvent describes an event where a contract has become available on the TestChain, either
// due to contract creation, or a self-destruct operation being reverted.
type ContractDeploymentsAddedEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Contract defines information for the contract which was deployed to the Chain.
	Contract *types.DeployedContractBytecode
}

// ContractDeploymentsRemovedEvent describes an event where a contract has become unavailable on the TestChain, either
// due to the reverting of a contract creation, or a self-destruct operation.
type ContractDeploymentsRemovedEvent struct {
	// Chain refers to the TestChain which emitted the event.
	Chain *TestChain

	// Contract defines information for the contract which was deployed to the Chain.
	Contract *types.DeployedContractBytecode
}
