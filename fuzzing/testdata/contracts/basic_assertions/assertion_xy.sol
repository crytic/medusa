contract TestAssertionXY {
    uint x;
    uint y;

    function setX(uint value) public {
        x = value + 3;
    }

    function setY(uint value) public {
        y = value + 9;

        // ASSERTION: x should never be 10 at the same time y is 80
        assert(!(x == 10 && y == 80));
    }


    function fuzz_not_solvable() public view returns (bool) {
        // ASSERTION: We put a dummy assertion here that will never fail.
        return true;
    }
}
