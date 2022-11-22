// This contract ensures the fuzzer can solve a magic number problem and report an assertion failure.
contract TestContract {
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
}
