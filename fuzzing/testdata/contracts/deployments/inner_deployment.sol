// InnerDeploymentFactory deploys InnerDeployment when a method is called after deployment, and verifies the fuzzer can
// match bytecode and fail the test appropriately.
contract InnerDeployment {
    function property_inner_deployment() public view returns (bool) {
        // ASSERTION: Fail immediately.
        return false;
    }
}

contract InnerDeploymentFactory {
    function deployInner() public returns (address) {
        return address(new InnerDeployment());
    }
}
