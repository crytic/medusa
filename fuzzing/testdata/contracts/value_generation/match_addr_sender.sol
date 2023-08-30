// This contract verifies the fuzzer can guess the msg.sender address as a function input.
contract TestContract {
    address a;
    address sender;

    function setAddr(address value) public {
        a = value;
        sender = msg.sender;
    }

    function property_never_specific_values() public view returns (bool) {
        // ASSERTION: a should not be sender's address who set it.
        return a != sender;
    }
}
