// InnerInnerDeployment takes constructor arguments so it will not be deployed automatically by the fuzzer.
// We use InnerDeployment to deploy InnerInnerDeployment and verify the fuzzer detects this appropriately.
contract InnerInnerDeployment {
    uint x;

    // We add a constructor here so it's not automatically deployed, this way we test dynamic deployment.
    constructor(uint dummyValue) public {
        x = dummyValue;
    }

    function fuzz_inner_inner_deployment() public view returns (bool) {
        // ASSERTION: Fail immediately.
        return false;
    }
}

// InnerDeployment takes constructor arguments so it will not be deployed automatically by the fuzzer.
// We use InnerDeploymentFactory to deploy InnerDeployment and verify the fuzzer detects this appropriately.
contract InnerDeployment {
    uint x;

    // We add a constructor here so it's not automatically deployed, this way we test dynamic deployment.
    constructor(uint dummyValue) public {
        x = dummyValue;
    }

    function deployInnerInner() public returns (address) {
        return address(new InnerInnerDeployment(7));
    }
}

// InnerDeploymentFactory deploys InnerDeployment to test inner deployments.
contract InnerDeploymentFactory {
    function deployInner() public returns (address) {
        return address(new InnerDeployment(7));
    }
}
