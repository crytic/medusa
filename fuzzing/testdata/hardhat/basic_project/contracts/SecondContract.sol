pragma solidity ^0.7.1;

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

    function fuzz_never_specific_values() public view returns (bool) {
        // ASSERTION: a should never be 10 at the same time b is 80 at the same time c is 14
        return !(a == 10 && b == 80 && c == 14);
    }
}
