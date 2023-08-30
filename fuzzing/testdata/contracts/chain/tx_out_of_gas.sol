// This contract verifies the fuzzer behaves well when the only transactions that can be added are ones which run out
// of gas.
contract TestContract {
    uint x;
    function useAllGas() public {
        for(uint i = 0; true; i++) {
            x = i; // SSTORE is expensive, will simulate failure faster
            x = i * 2;
            x = i;
        }
    }

    function property_never_apply_state_when_oog() public view returns (bool) {
        // ASSERTION: this state should never be applied, as our out of gas error should revert changes.
        return x == 0;
    }
}
