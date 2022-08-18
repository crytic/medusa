contract TestXY {
    uint x;
    uint y;

    function setX(uint value) public {
        x = value + 3;
    }

    function setY(uint value) public {
        y = value + 9;
    }


    function medusa_never_specific_values() public view returns (bool) {
        // ASSERTION: x should never be 10 at the same time y is 80
        return !(x == 10 && y == 80);
    }
}