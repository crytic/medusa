// This test ensures that the expectEmit cheatcode works as expected
interface CheatCodes {
    function expectEmit() external;

    function expectEmit(
        bool checkTopic1,
        bool checkTopic2,
        bool checkTopic3,
        bool checkData
    ) external;

    function expectEmit(address emitter) external;

    function expectEmit(
        bool checkTopic1,
        bool checkTopic2,
        bool checkTopic3,
        bool checkData,
        address emitter
    ) external;

    function difficulty(uint256) external;
}

contract Token {
    event Transfer(address indexed from, address indexed to, uint256 amount);

    function transfer(address to, uint256 amount) public {
        emit Transfer(msg.sender, to, amount);
    }
}

contract TestContract {
    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(
            0x7109709ECfa91a80626fF3989D68f67F5b1DD12D
        );

        // Deploy a token contract.
        Token token = new Token();

        cheats.expectEmit();
        emit Token.Transfer(address(this), address(1), 10);
        token.transfer(address(1), 10);

        cheats.expectEmit(true, true, false, true);
        emit Token.Transfer(address(this), address(1), 10);
        token.transfer(address(1), 10);

        cheats.expectEmit(address(token));
        emit Token.Transfer(address(this), address(1), 10);
        token.transfer(address(1), 10);

        cheats.expectEmit(true, true, false, true, address(token));
        emit Token.Transfer(address(this), address(1), 10);
        token.transfer(address(1), 10);
    }
}
