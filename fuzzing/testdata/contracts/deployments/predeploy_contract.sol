contract PredeployContract {
    bool called = false;

    function triggerFailure() public {
        if (called) {
            assert(false);
        }
        called = true;
    }
}

contract TestContract {
    PredeployContract predeploy = PredeployContract(address(0x1234));

    function testPredeploy() public {
        predeploy.triggerFailure();
    }
}
