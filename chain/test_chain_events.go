package chain

import "github.com/trailofbits/medusa/chain/types"

// BlockAddedEvent describes an event where a new block is added to the TestChain. This only considers internally
// committed blocks, not ones spoofed in between block number jumps.
type BlockAddedEvent struct {
	Chain *TestChain
	Block *types.Block
}

// BlockRemovedEvent describes an event where a block is removed from the TestChain. This only considers internally
// committed blocks, not ones spoofed in between block number jumps.
type BlockRemovedEvent struct {
	Chain *TestChain
	Block *types.Block
}

// ContractDeploymentsAddedEvent describes an event where a new contract deployments are detected by the TestChain.
type ContractDeploymentsAddedEvent struct {
	Chain                     *TestChain
	DeployedContractBytecodes []*types.DeployedContractBytecode
}

// ContractDeploymentsRemovedEvent describes an event where a previously deployed contract on the TestChain is removed,
// possibly due to the chain reverting to a previous block or a self-destruct operation.
type ContractDeploymentsRemovedEvent struct {
	Chain                     *TestChain
	DeployedContractBytecodes []*types.DeployedContractBytecode
}
