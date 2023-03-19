// This contract ensures the fuzzer's execution tracing can parse contract creation.
contract InnerDeploymentContract {
    uint x;
    constructor(uint y, string memory message) public {
        x = y;
    }
}

contract TestContract {
    function deployInner() public returns (address) {
        address a = address(new InnerDeploymentContract(7, "Hello from deployment args!"));
        assert(false);
    }
}
