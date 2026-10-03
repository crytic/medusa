// This contract tests sometimes assertions across multiple contracts.
contract TestContract {
    uint256 public state;

    function setState(uint256 x) public {
        state = x;
    }

    // Should PASS - state gets set
    function sometimes_stateSet() public view {
        require(state > 0, "State should be set sometimes");
    }
}

contract SecondContract {
    bool public flag;

    function setFlag() public {
        flag = true;
    }

    // Should PASS - flag gets set
    function sometimes_flagSet() public view {
        require(flag, "Flag should be set sometimes");
    }
}
