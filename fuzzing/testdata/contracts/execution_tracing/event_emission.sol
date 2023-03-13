// This contract ensures the fuzzer can detect assertions by solving a simple problem.
contract TestContract {
    uint x;
    uint y;

    event TestEvent(uint value);
    event TestIndexedEvent(uint indexed value);
    event TestMixedEvent(address indexed sender, uint x, int32 indexed y, string s);

    function setX(uint value) public {
        x = value;

        // ASSERTION: x should be an even number

        emit TestEvent(value);
        emit TestIndexedEvent(value);
        emit TestMixedEvent(msg.sender, value, 7, "Hello from an event emission!");
        assert(x % 2 == 0);
    }
}
