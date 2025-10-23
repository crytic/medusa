// This test ensures that label can be set for an address
interface CheatCodes {
    function label(address, string memory) external;
}

// This contract is the implementation contract.
contract ImplementationContract {
    event TestEvent(address);

    function emitEvent(address addr) public returns(address) {
        // We are emitting this event to see if emitting the random address will capture the label for it
        emit TestEvent(address(0x20000));

        // We return an address to see if the label is captured
        return addr;
    }
}

// This contract tests the label cheatcode. We use a delegatecall because it ensures that all possible cases are tested
// for the execution trace. We also provide an address as an input argument, return value, and event argument to see
// if the label works properly for those as well.
contract TestContract {
    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    // Reference to our implementation contract
    ImplementationContract impl;

    constructor() public {
        // Deploy our implementation contract
        impl = new ImplementationContract();
        // Label this contract
        cheats.label(address(this), "ProxyContract");
        // Label the sender
        cheats.label(address(0x10000), "MySender");
        // Label the implementation contract
        cheats.label(address(impl), "ImplementationContract");
        // Label a random address
        cheats.label(address(0x20000), "RandomAddress");
    }

    function testVMLabel() public {
        // Perform a delegate call
        (bool success, bytes memory data) = address(impl).delegatecall(abi.encodeWithSignature("emitEvent(address)", address(this)));

        // Trigger an assertion failure to capture the execution trace
        assert(false);
    }
}

