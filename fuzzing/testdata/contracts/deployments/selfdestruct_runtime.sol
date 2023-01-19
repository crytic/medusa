// TestContract deploys an InnerDeployment contract upon construction. InnerDeployment provides a method which will
// trigger a selfdestruct. This is used to test contract existence checks.
contract InnerDeployment {
    function destroy() public {
        selfdestruct(payable(address(0)));
    }
}

contract InnerDeploymentFactory {
    address a;

    constructor() {
        a = address(new InnerDeployment());
    }

    function dummyFunction(uint x) public {
        // This exists so the fuzzer knows there are state changing methods to target, instead of quitting early.
        x = 7;
    }
}
