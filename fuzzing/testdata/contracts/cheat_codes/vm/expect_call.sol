// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

interface CheatCodes {
    function expectCall(address where, bytes calldata data) external;

    function expectCall(
        address where,
        bytes calldata data,
        uint64 count
    ) external;

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
    event Transfer(
        address indexed from,
        address indexed to,
        uint256 indexed amount
    );

    function deposit(uint256 amount) external payable {
        assert(false);
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
        CheatCodes cheats = CheatCodes(
            0x7109709ECfa91a80626fF3989D68f67F5b1DD12D
        );
        Bank bank = new Bank();

        // Expect a call to bank.transfer with specific calldata
        cheats.expectCall(address(bank), abi.encodeCall(bank.transfer, (to, amount)));
        bank.transfer(to, amount);

        // Expect a call to bank.transfer with any calldata, 2 times
        cheats.expectCall(address(bank), abi.encodeWithSelector(bank.transfer.selector, to, amount), 2);
        bank.transfer(to, amount);
        bank.transfer(to, amount);

        // Expect a call to bank.transfer with specific calldata, 4 times
        cheats.expectCall(address(bank), abi.encodeWithSelector(bank.transfer.selector, to, amount), 4);
        bank.transfer(to, amount);
        bank.transfer(to, amount);
        bank.transfer(to, amount);
        bank.transfer(to, amount);

        //        // Expect a call to bank.deposit with specific value and calldata
//        cheats.expectCall(address(bank), 3, abi.encodeCall(bank.deposit, (3)));
//        bank.deposit{value: value}(value);

//        // Expect a call to bank.deposit with specific value and calldata, 5 times
//        cheats.expectCall(address(bank), value, abi.encodeCall(bank.deposit, ()), 5);
//        bank.deposit{value: value}();
//        bank.deposit{value: value}();
//        bank.deposit{value: value}();
//        bank.deposit{value: value}();
//        bank.deposit{value: value}();
    }
}
