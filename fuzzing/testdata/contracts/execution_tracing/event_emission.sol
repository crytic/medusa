// This test ensures the fuzzer's execution tracing can obtain event emissions.
library Logger {
    event TestLibraryEvent(string s);

    function log(string memory s) internal {
        emit TestLibraryEvent(s);
    }
}

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
        Logger.log("Hello from library event args!");
        assert(x % 2 == 0);
    }
}
