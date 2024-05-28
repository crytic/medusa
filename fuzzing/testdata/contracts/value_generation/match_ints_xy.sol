// This contract verifies two different, but specific int function arguments will be provided by the fuzzer.
contract TestContract {
    int x;
    int y;

    function setX(int value) public {
        x = value + 3;
    }

    function setY(int value) public {
        y = value + 9;
    }


    function property_never_specific_values() public view returns (bool) {
        // ASSERTION: x should never be -10 at the same time y is -62
        return !(x == -10 && y == -62);
    }
}
