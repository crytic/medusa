# `getCode`

## Description
The `getCode` cheatcode returns the creation bytecode for a contract in the project given the path to the contract.

## Example

```
contract TestContract {
    function test() public {
        // Obtain cheat code contract reference
        IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Get the creation bytecode for a contract
        bytes memory bytecode = cheats.getCode("MyContract.sol");
        
        // Deploy the contract using the bytecode
        address deployed;
        assembly {
            deployed := create(0, add(bytecode, 0x20), mload(bytecode))
        }
        
        // Verify the contract was deployed successfully
        require(deployed != address(0), "Deployment failed");
    }
}
```


## Function Signature

```solidity
function getCode(string calldata) external returns (bytes memory);
```