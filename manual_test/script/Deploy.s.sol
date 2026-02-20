// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

interface Vm {
    function startBroadcast() external;
    function stopBroadcast() external;
}

contract DeployScript {
    Vm constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    function run() external {
        vm.startBroadcast();

        // Deploy counter with initial value
        Counter counter = new Counter(42);

        // Perform some initial operations to create interesting state
        counter.increment();
        counter.increment();
        counter.increment();

        vm.stopBroadcast();
    }
}

contract Counter {
    uint256 public count;

    constructor(uint256 initialCount) {
        count = initialCount;
    }

    function increment() public {
        count++;
    }

    function decrement() public {
        require(count > 0, "Count cannot be negative");
        count--;
    }

    // Property: count should never overflow
    function echidna_count_reasonable() public view returns (bool) {
        return count < type(uint256).max / 2;
    }
}
