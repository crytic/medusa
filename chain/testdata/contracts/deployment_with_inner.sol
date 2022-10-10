
contract InnerDeployment {
    uint x;

    constructor(uint dummyValue) public {
        x = dummyValue;
    }

    function reset() public {
        x = 0;
    }
}

contract InnerDeploymentFactory {
    InnerDeployment x;
    constructor() public {
        x = new InnerDeployment(7);
        x.reset();
    }
}
