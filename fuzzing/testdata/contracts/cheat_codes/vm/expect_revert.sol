// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

// This test ensures that a revert can be expected from the next call
interface CheatCodes {
    function expectRevert() external;
    function expectRevert(string memory) external;
}

contract BankContract {
    function send(uint256 amount) public view {
        require(amount > 0, "amount must be greater than 0");
    }
}

contract TestContract {
    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        BankContract bank = new BankContract();

        cheats.expectRevert();
        bank.send(0);

        cheats.expectRevert("amount must be greater than 0");
        bank.send(0);
    }
}
