// This contract is used to test deployment of contracts with constructor arguments.
contract DeploymentWithArgs {
    uint x;
    bytes y;

    constructor(uint _x, bytes memory _y) {
        x = _x;
        y = _y;
    }
}
