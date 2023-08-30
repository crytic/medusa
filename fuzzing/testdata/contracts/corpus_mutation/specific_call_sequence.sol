// This contract verifies the fuzzer can use corpus mutations and coverage guiding to solve a simple problem of needing
// to provide specific input sequentially with calls.
contract TestContract {
    uint index;
    bool permanentFail;

    function checkValue(uint x) public {
        // If we called in the wrong order previously, this contract is permanently locked, so it can never pass.
        require(!permanentFail);

        // If we called in the correct order with the correct value, proceed to the next index.
        // Otherwise we lock the contract permanently.
        if (index == 0 && x == 12) {
            index++;
        } else if (index == 1 && x == 23) {
            index++;
        } else if (index == 2 && x == 34) {
            index++;
        } else if (index == 3 && x == 45) {
            index++;
        } else if (index == 4 && x == 56) {
            index++;
        } else if (index == 5 && x == 67) {
            index++;
        } else if (index == 6 && x == 78) {
            index++;
        } else {
            permanentFail = true;
        }
    }

    function property_solve_me() public view returns (bool) {
        // ASSERTION: The fuzzer should be able to fail this test case and solve all challenges.
        return index < 7;
    }
}
