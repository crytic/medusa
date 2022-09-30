
contract InnerDeployment {
    uint x;

    constructor(uint dummyValue) {
        x = dummyValue;
    }

    function reset() public {
        x = 0;
    }
}

contract InnerDeploymentFactory {
    InnerDeployment x;
    constructor() {
        x = new InnerDeployment(7);
        x.reset();
    }
}
