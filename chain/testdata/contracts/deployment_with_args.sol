pragma solidity ^0.6.6;

// This contract is used to test deployment of contracts with constructor arguments.
contract DeploymentWithArgs {
    uint x;
    bytes y;

    constructor(uint _x, bytes memory _y) public {
        x = _x;
        y = _y;
    }
}
