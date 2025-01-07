contract TestContract {
    constructor() {}

    function testFuzz(uint256 z) public {
        uint256 x = 123;
        int256 y = 5 * 72;
        address addr = address(0);
        string memory str = "hello world";
        assert(false);
    }
}