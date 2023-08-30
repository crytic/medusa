// TestContract deploys a TestContractChild on construction, both containing failing assertion and property tests,
// to ensure that the project configuration surrounding the "test all contracts" feature works as expected for
// dynamically deployed contracts.
contract TestContractChild {
    function failing_assertion_method_child(uint x) public {
        assert(false);
    }

    function property_failing_property_test_method_child() public view returns (bool) {
        return false;
    }
}

contract TestContract {
    address a;

    constructor() public {
        a = address(new TestContractChild());
    }

    function failing_assertion_method(uint x) public {
        assert(false);
    }

    function property_failing_property_test_method() public view returns (bool) {
        return false;
    }
}
