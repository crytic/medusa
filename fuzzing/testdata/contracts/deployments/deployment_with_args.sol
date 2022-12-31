// This contract is used to test deployment of contracts with constructor arguments.
contract DeploymentWithArgs {
    struct Abc {
        uint a;
        bytes b;
    }

    uint x;
    bytes2 y;
    Abc z;

    constructor(uint _x, bytes2 _y, Abc memory _z) {
        x = _x;
        y = _y;
        z = _z;
    }

    function fuzz_checkX() public returns (bool) {
        return x != 123456789;
    }

    function fuzz_checkY() public returns (bool) {
        return y != 0x5465;
    }

    function fuzz_checkZ() public returns (bool) {
        return z.a != 0x4d2;
    }

    function dummyFunction(uint a) public {
        // This exists so the fuzzer knows there are state changing methods to target, instead of quitting early.
        a = 8;
    }
}

contract Dependent {
    address deployed;

    constructor(address _deployed) {
        deployed = _deployed;
    }

    function fuzz_checkDeployed() public returns (bool) {
        return deployed == 0x0000000000000000000000000000000000000000;
    }

    function dummyFunction(uint a) public {
        // This exists so the fuzzer knows there are state changing methods to target, instead of quitting early.
        a = 8;
    }
}