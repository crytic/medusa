// This contract is used to test deployment of contracts with constructor arguments.
contract DeploymentWithArgs {
    uint x;
    bytes y;

    constructor(uint _x, bytes memory _y) {
        x = _x;
        y = _y;
    }

    function fuzz_checkX() public returns (bool) {
        return x == 0;
    }

    function fuzz_checkY() public returns (bool) {
        return y.length == 0;
    }

    function dummyFunction(uint a) public {
        // This exists so the fuzzer knows there are state changing methods to target, instead of quitting early.
        a = 8;
    }
}