import "./SimpleContract.sol";

contract DerivedContract is SimpleContract {
    uint z;

    function setZ(uint value) public {
        z = value;
    }
}
