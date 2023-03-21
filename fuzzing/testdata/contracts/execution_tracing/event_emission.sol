// This contract ensures the fuzzer's execution tracing can obtain event emissions.
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
        emit TestMixedEvent(msg.sender, value, 7, "Hello from event args!");
        assert(x % 2 == 0);
    }
}
