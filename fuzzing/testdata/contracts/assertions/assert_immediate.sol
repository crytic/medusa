pragma solidity ^0.8.0;

// This contract includes a function that we will call from TestContract.
contract AnotherContract {
    // This function doesn't need to do anything specific for this example.
    function externalFunction() public pure returns (string memory) {
        return "External function called";
    }
}

// This contract ensures the fuzzer can encounter an immediate assertion failure.
contract TestContract {
    AnotherContract public anotherContract;

    event ValueReceived(uint indexed value, uint second_val);
    event ValueNonIndexedReceived(uint firstval, uint secondval);

    function internalFunction() public returns (string memory) {
        anotherContract.externalFunction();
        return "Internal function called";
    }

    // Deploy AnotherContract within the TestContract
    constructor() {
//        internalFunction();
        anotherContract = new AnotherContract();
    }


    function callingMeFails(uint value) public {
        // Call internalFunction()
        internalFunction();
        // Call the external function in AnotherContract.
        anotherContract.externalFunction();

        uint second_val = 2+12;

        emit ValueReceived(value, second_val);
        emit ValueNonIndexedReceived(111+111, 444+444);

        // ASSERTION: We always fail when you call this function.
        assert(false);

    }
}
