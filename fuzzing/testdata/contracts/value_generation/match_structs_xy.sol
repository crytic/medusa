// This contract verifies the fuzzer can provide a struct with exact expected values in its structure.

pragma experimental ABIEncoderV2;

struct TestStructInner {
    uint y;
    bool b;
}

struct TestStruct {
    uint x;
    uint y;
    string s;
    TestStructInner i;
}

contract TestContract {
    uint x;
    uint y;
    TestStruct s;

    function setStruct(TestStruct memory ts) public {
        s = ts;
    }

    function property_never_specific_values() public view returns (bool) {
        // ASSERTION: x should never be 10 at the same time y is 80
        return !(s.x == 10 && s.i.y == 80);
    }
}
