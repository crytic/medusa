// This contract ensures the fuzzer can detect assertions by solving a simple problem.
contract TestContract {
    uint x;
    uint y;

    event TestEvent(uint value);
    event TestIndexedEvent(uint indexed value);

    function setX(uint value) public {
        x = value;

        // ASSERTION: x should be an even number

        emit TestEvent(value);
        emit TestIndexedEvent(value);
        assert(x % 2 == 0);
    }
}
