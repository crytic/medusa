pragma solidity ^0.8.10;

contract FirstContract {
    uint x;

    function setX(uint value) public {
        x = value + 3;
    }
}

contract InheritedFirstContract is FirstContract {
    uint y;

    function setY(uint value) public {
        y = value + 9;
    }

    function fuzz_never_specific_values() public view returns (bool) {
        // ASSERTION: x should never be 10 at the same time y is 80
        return !(x == 10 && y == 80);
    }
}
