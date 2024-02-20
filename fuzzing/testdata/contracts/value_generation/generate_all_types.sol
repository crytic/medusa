// This contract contains functions with various input argument types to test value generation in the fuzzer.
contract GenerateAllTypes {
    uint x;
    int y;
    bytes b = "";

    function setUint(uint value) public {
        x = value + 3;
    }

    // TODO: Uint types of various sizes

    function setInt(int value) public {
        y = value + 0;
    }

    // TODO: Int types of various sizes

    function setString(string memory s) public {
        s = "";
    }

    function setBytes(bytes memory s) public {
        s = "";
    }

    function property_never_fail() public view returns (bool) {
        // ASSERTION: never fail, to keep testing value generation
        return true;
    }
}
