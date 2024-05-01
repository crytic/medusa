contract PredeployContract {
    function triggerFailure() public {
        assert(false);
    }
}

contract TestContract {
    PredeployContract predeploy = PredeployContract(address(0x1234));

    function testPredeploy() public {
        predeploy.triggerFailure();
    }
}
