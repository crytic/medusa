library InternalLibrary {
    function double(uint x) internal pure returns (uint, uint) {
        return (x, x);
    }
}

contract TestInternalLibrary {
    using InternalLibrary for uint;
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
        (a, b) = InternalLibrary.double(x);
        if (a != x || b != x) {
            failedTest = true;
        }
        return a + b;
    }

    function property_library_linking_broken() public view returns (bool) {
        // ASSERTION: We should always be able to compute correctly.
        return !failedTest;
    }
}
