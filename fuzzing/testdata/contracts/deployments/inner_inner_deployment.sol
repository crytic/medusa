// InnerDeploymentFactory deploys InnerDeployment when a method is called after deployment. After InnerDeployment is
// deployed, a method can be used to deploy an InnerInnerDeployment. We verify we can violate an invariant
// in a two-layer deep dynamic deployment.
contract InnerInnerDeployment {
    function property_inner_inner_deployment() public view returns (bool) {
        // ASSERTION: Fail immediately.
        return false;
    }
}

contract InnerDeployment {
    function deployInnerInner() public returns (address) {
        return address(new InnerInnerDeployment());
    }
}

// TestContract deploys InnerDeployment to test inner deployments.
contract InnerDeploymentFactory {
    function deployInner() public returns (address) {
        return address(new InnerDeployment());
    }
}
