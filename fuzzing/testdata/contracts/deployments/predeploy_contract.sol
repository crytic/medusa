contract PredeployContract {
    function triggerFailure() public {
        assert(false);
    }
}

contract TestContract {
    PredeployContract predeploy = PredeployContract(address(0x123));

    function testPredeploy() public {
        predeploy.triggerFailure();
        assert(false);
    }
}