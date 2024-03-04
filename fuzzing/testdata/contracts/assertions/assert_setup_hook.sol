// These contracts ensure that setUp hooks work as expected.
contract TestContract {
    bool public state = false;

    function setUp() public {
        state = true;
    }

    function one() public {
        assert(state);

        state = false;
    }

    function two() public {
        assert(state);

        state = false;
    }

    function three() public {
        assert(state);

        state = false;
    }
}

contract TestContract2 {
    uint256 public num = 0;

    function setUp() public {
        num = 3;
    }

    function four() public {
        assert(num == 3);
    }

    function five() public {
        assert(num == 3);
    }

    function six() public {
        assert(num == 3);
    }
}