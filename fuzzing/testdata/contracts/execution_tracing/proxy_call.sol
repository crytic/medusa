// This contract ensures the fuzzer's execution tracing can trace proxy calls appropriately.
contract InnerDeploymentContract {
    uint x;
    uint y;
    function setXY(uint _x, uint _y, string memory s) public payable {
        x = _x;
        y = _y;
    }
}

contract TestContract {
    uint x;
    uint y;
    InnerDeploymentContract i;

    constructor() public {
        i = new InnerDeploymentContract();
    }

    function testDelegateCall() public returns (address) {
        // Perform a delegate call to set our variables in this contract.
        (bool success, bytes memory data) = address(i).delegatecall(abi.encodeWithSignature("setXY(uint256,uint256,string)", 123, 321, "Hello from proxy call args!"));

        // Trigger an assertion failure (ending the test), if data was set successfully.
        // If we don't hit this assertion failure, something is wrong.
        assert(!(x == 123 && y == 321));
    }
}
