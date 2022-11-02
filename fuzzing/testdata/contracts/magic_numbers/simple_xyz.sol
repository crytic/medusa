contract TestMagicNumbersXY {
    uint x;
    uint y;

    function setX(uint value) public {
        x = value + 3;
    }

    function setY(uint value) public {
        y = value + 9;
    }

    function setZ(uint value) public {
        z = value + 7;
    }

    function fuzz_never_specific_values() public view returns (bool) {
        // ASSERTION: x should never be 10 at the same time y is 80 at the same time z is 14
        return !(x == 10 && y == 80 && z == 14);
    }
}
