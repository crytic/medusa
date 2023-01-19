// This contract verifies that the slither printer is working as expected to solve properties like this
contract TestContract {
    uint x;

    function f(uint i) public payable {
        x = i;
    }

    function fuzz_ether() public view returns (bool) {
        return x != 2 * 2 ether;
    }
}