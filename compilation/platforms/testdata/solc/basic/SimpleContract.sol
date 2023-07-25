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

contract SimpleContract2 {
    uint x;
    uint y;

    function setX(uint value) public returns (bool) {
        x = value;
        return true;
    }

    function setY(uint value) public {
        y = value;
    }
}