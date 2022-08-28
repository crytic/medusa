contract SimpleContract {
    uint x;
    uint y;

    function setX(uint value) public {
        x = value;
    }

    function setY(uint value) public {
        y = value;
    }
}

contract InheritedContract is SimpleContract {
    uint z;

    function setZ(uint value) public {
        z = value;
    }
}
