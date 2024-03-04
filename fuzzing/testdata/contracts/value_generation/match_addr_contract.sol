// This contract verifies the fuzzer can guess the contract's address itself.
contract TestContract {
    address a;

    function setAddr(address value) public {
        a = value;
    }

    function property_never_specific_values() public view returns (bool) {
        // ASSERTION: a should not be the contract's address itself.
        return !(a == address(this));
    }
}
