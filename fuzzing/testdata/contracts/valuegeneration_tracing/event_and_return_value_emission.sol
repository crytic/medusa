pragma solidity ^0.8.0;

contract AnotherContract {
    // This function returns a variety of values that need to be captured in the value set
    function testAnotherFunction(uint256 x) public pure returns (uint256, int256, string memory, address, bytes memory, bytes4) {
        // Fix the values we want to return
        uint256 myUint = 3;
        int256 myInt = 4;
        string memory myStr = "another string";
        address myAddr = address(0x5678);
        bytes memory myBytes = "another byte array";
        bytes4 fixedBytes = "word";

        return (myUint, myInt, myStr, myAddr, myBytes, fixedBytes);
    }
}

contract TestContract {
    AnotherContract public anotherContract;

    event EventValues(uint indexed myUint, int myInt, string myStr, address myAddr, bytes myBytes, bytes4 fixedBytes);

    // Deploy AnotherContract within the TestContract
    constructor() {
        anotherContract = new AnotherContract();
    }

    function testFunction(uint x) public {
        // Fix the values we want to emit
        uint256 myUint = 1;
        int256 myInt = 2;
        string memory myStr = "string";
        address myAddr = address(0x1234);
        bytes memory myBytes = "byte array";
        bytes4 fixedBytes = "byte";

        // Call an external contract
        anotherContract.testAnotherFunction(x);

        // Emit an event in this call frame
        emit EventValues(myUint, myInt, myStr, myAddr, myBytes, fixedBytes);

        // ASSERTION: We always fail when you call this function.
        assert(false);

    }
}
