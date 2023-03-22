// This contract ensures the fuzzer's execution tracing can obtain custom errors

// Define a custom error type
error CustomError(string message, uint value);

// Define the contract to return a custom error.
contract ErrorContract {
    function returnErrorIfSeven(uint x) public {
        if (x == 7) {
            revert CustomError("Hello from a custom error!", x);
        }
    }
}

contract TestContract {
    ErrorContract rc;
    constructor() {
        rc = new ErrorContract();
    }

    function errorAndAssertFailIfSeven(uint value) public {
        try rc.returnErrorIfSeven(value) {
            return;
        } catch Error(string memory /*reason*/) {
            assert(false);
        } catch (bytes memory /*lowLevelData*/) {
            assert(false);
        }
    }
}
