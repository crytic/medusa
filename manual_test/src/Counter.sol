// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

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
