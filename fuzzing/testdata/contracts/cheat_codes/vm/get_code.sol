// This test ensures that contract creation bytecode can be fetched with cheat codes
interface CheatCodes {
    function getCode(string calldata) external returns (bytes memory);
}

// Define a simple contract to deploy and retrieve code from
contract SimpleStorage {
    uint256 private value;
    
    function set(uint256 newValue) public {
        value = newValue;
    }
    
    function get() public view returns (uint256) {
        return value;
    }
}

// Test contract to verify getCode functionality
contract TestGetCode {
    function testGetCode() public {
        // Obtain our cheat code contract reference
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        
        // Get bytecode for SimpleStorage contract
        bytes memory bytecode = cheats.getCode("SimpleStorage");
        
        // Verify we got bytecode
        require(bytecode.length > 0, "Failed to get contract bytecode");
        
        // Deploy the contract using the retrieved bytecode
        address deployedAddr;
        assembly {
            deployedAddr := create(0, add(bytecode, 0x20), mload(bytecode))
        }
        
        // Verify deployment was successful
        require(deployedAddr != address(0), "Failed to deploy contract using retrieved bytecode");
        
        // Test that we can interact with the deployed contract
        SimpleStorage simpleStorage = SimpleStorage(deployedAddr);
        simpleStorage.set(22);
        require(simpleStorage.get() == 22, "Contract functionality doesn't work correctly");
    }        
}