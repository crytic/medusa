pragma solidity ^0.8.10;

contract FirstContract {
    uint x;
    uint y;

    function setX(uint value) public {
        x = value;
    }

    function setY(uint value) public {
        y = value;
    }
}

contract InheritedFirstContract is FirstContract {
    uint z;

    function setZ(uint value) public {
        z = value;
    }
}