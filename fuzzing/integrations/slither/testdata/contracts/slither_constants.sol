contract SlitherConstants {
    uint x;

    function f(uint i) public payable {
        x = i;
    }

    function echidna_uint() public view returns (bool) {
        return x != uint(-2);
    }

    function echidna_ether() public view returns (bool) {
        return x != 2 * 2 ether;
    }
}