pragma solidity ^0.7.1;

contract SecondContract {
    uint x;
    uint y;

    function setX(uint value) public {
        x = value;
    }

    function setY(uint value) public {
        y = value;
    }
}

contract InheritedSecondContract is SecondContract {
    uint z;

    function setZ(uint value) public {
        z = value;
    }
}
