// This contract ensures the fuzzer's execution tracing can obtain revert reasons
contract RevertingContract {
    function revertIfSeven(uint x) public {
        if (x == 7) {
            revert("RevertingContract was called and reverted.");
        }
    }
}

contract TestContract {
    RevertingContract rc;
    constructor() {
        rc = new RevertingContract();
    }

    function assertIfRevertEncountered(uint value) public {
        try rc.revertIfSeven(value) {
            return;
        } catch Error(string memory /*reason*/) {
            assert(false);
        }
    }
}
