// This contract ensures the fuzzer's execution tracing can trace self destruct operations
contract SelfDestructContract {
    address owner;
    constructor() {
        owner = msg.sender;
    }

    function destroyIfOwner() public {
        require(msg.sender == owner);
        selfdestruct(payable(address(0)));
    }
}

contract TestContract {
    SelfDestructContract sdc;
    constructor() {
        sdc = new SelfDestructContract();
    }

    function destroyContractAndTriggerFailure(uint value) public {
        sdc.destroyIfOwner();
        assert(false);
    }
}
