// This contract verifies the fuzzer can guess an exact string as a function input.
contract TestContract {
    string MAGIC_STRING = "anExactString";
    string s;
    function setString(string memory value) public {
        s = value;
    }

    function property_never_specific_values() public view returns (bool) {
        // ASSERTION: s should not be the MAGIC_STRING
        return keccak256(abi.encodePacked((s))) != keccak256(abi.encodePacked((MAGIC_STRING)));
    }
}
