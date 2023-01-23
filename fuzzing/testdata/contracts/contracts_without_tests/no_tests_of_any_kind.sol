// This contract ensures the fuzzer do not run if no tests of any kind exist.
contract NoTests {
    string public message;

    constructor() public {
        message = "Hello, World!";
    }
}
