// InnerDeploymentFactory deploys a InnerDeployment on construction and verifies the fuzzer can match bytecode and
// fail the test appropriately.
contract InnerDeployment {
    function dummyFunction(uint x) public {
        // This exists so the fuzzer knows there are state changing methods to target, instead of quitting early.
        x = 7;
    }

    function property_inner_deployment() public view returns (bool) {
        // ASSERTION: Fail immediately.
        return false;
    }
}

contract InnerDeploymentFactory {
    address a;

    constructor() public {
        a = address(new InnerDeployment());
    }

    function dummyFunction(uint x) public {
        // This exists so the fuzzer knows there are state changing methods to target, instead of quitting early.
        x = 8;
    }
}
