library ExternalLibrary {
    function double(uint x) external pure returns (uint, uint) {
        return (x, x);
    }
}

contract TestExternalLibrary {
    using ExternalLibrary for uint;
    bool failedTest;

    function testExtension(uint x, uint y) public returns (uint) {
        uint a;
        uint b;
        (a, b) = x.double();
        if (a != x || b != x) {
            failedTest = true;
        }
        return a + b;
    }

    function testDirect(uint x) public returns (uint) {
        uint a;
        uint b;
        (a, b) = ExternalLibrary.double(x);
        if (a != x || b != x) {
            failedTest = true;
        }
        return a + b;
    }

    function fuzz_library_linking() public view returns (bool) {
        // ASSERTION: We should always be able to compute correctly.
        return !failedTest;
    }
}
