// This contract ensures the fuzzer can detect assertions by solving a simple problem.
contract TestContract {
    uint x;
    uint y;

    function setX(uint value) public {
        x = value;

        // ASSERTION: x should be an even number
        assert(x % 2 == 0);
    }
}
