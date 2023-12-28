// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

interface CheatCodes {
    function expectCall(address where, bytes calldata data) external;
    function expectCall(address where, bytes calldata data, uint64 count) external;
    function expectCall(
        address where,
        uint256 value,
        bytes calldata data
    ) external;
    function expectCall(
        address where,
        uint256 value,
        bytes calldata data,
        uint64 count
    ) external;
}

contract Bank {
    event Deposit(uint256 value);
    event Transfer(address indexed from, address indexed to, uint256 indexed amount);

    function deposit() external payable {
        emit Deposit(msg.value);
    }

    function transfer(address to, uint256 amount) external {
        emit Transfer(msg.sender, to, amount);
    }
}

contract TestContract {
    function test(address to, uint256 amount, uint256 value) public {
        require(value > 0);

        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        Bank bank = new Bank();

//        cheats.expectCall(address(token), abi.encodeCall(token.transfer, (to, amount)));
        // make the call we expect
//        token.transfer(to, amount);

//        cheats.expectCall(address(token), abi.encodeCall(token.transfer, (to, amount)), 4);
//        token.transfer(to, amount);
//        token.transfer(to, amount);
//        token.transfer(to, amount);
//        token.transfer(to, amount);

        cheats.expectCall(address(bank), value, abi.encodeCall(bank.deposit, ()));
        bank.deposit{value: value}();

//        cheats.expectCall(address(bank), value, abi.encodeCall(bank.deposit, ()), 5);
//        bank.deposit{value: value}();
//        bank.deposit{value: value}();
//        bank.deposit{value: value}();
//        bank.deposit{value: value}();
//        bank.deposit{value: value}();
    }
}
