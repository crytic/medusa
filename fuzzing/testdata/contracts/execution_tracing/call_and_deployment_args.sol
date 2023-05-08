// This contract ensures the fuzzer's execution tracing can parse contract deployment args and call args.
contract InnerDeploymentContract {
    uint x;
    constructor(uint y, string memory message) public {
        x = y;
    }

    function callWithString(string memory message) public {
        assert(false);
    }
}

contract TestContract {
    function deployInner() public returns (address) {
        InnerDeploymentContract i = new InnerDeploymentContract(7, "Hello from deployment args!");
        i.callWithString("Hello from call args!");
    }
}
