// This contract verifies the fuzzer can guess exact addresses as a function input. We test hex and decimal derived
// addresses to ensure the AST seeding parses both correctly.
contract TestContract {
    address x;
    address y;

    function setX(address value) public {
        x = value;
    }

    function setY(address value) public {
        y = value;
    }

    function property_never_specific_values() public view returns (bool) {
        // ASSERTION: x and y should not equal the exact addresses below.
        return !(x == address(0x12345) && y == address(7));
    }
}
