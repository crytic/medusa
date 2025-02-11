contract PredeployContract {
    function triggerFailure() public {
        assert(false);
    }
}

contract TestContract {
    PredeployContract predeploy = PredeployContract(address(0x1234));

    constructor() payable {}
    
    function testPredeploy() public {
        predeploy.triggerFailure();
    }
}
