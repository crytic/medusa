// This source file provides two contracts which can be used to test deployment order and invariants that the fuzzer
// is expected to hold when modifying it.
contract FirstContract {
    uint x;

    function setX(uint value) public {
        x = value + 3;
    }
}

contract InheritedFirstContract is FirstContract {
    uint y;

    function setY(uint value) public {
        y = value + 9;
    }

    function property_never_specific_values() public view returns (bool) {
        // ASSERTION: x should never be 10 at the same time y is 80
        return !(x == 10 && y == 80);
    }
}

contract SecondContract {
    uint a;
    uint b;

    function setA(uint value) public {
        a = value + 3;
    }

    function setB(uint value) public {
        b = value + 9;
    }
}

contract InheritedSecondContract is SecondContract {
    uint c;

    function setC(uint value) public {
        c = value + 7;
    }

    function property_never_specific_values() public view returns (bool) {
        // ASSERTION: a should never be 10 at the same time b is 80 at the same time c is 14
        return !(a == 10 && b == 80 && c == 14);
    }
}