// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

contract Benchmark {
    uint256 private value;
    mapping(uint256 => uint256) private balances;

    function step(uint256 input) public returns (uint256) {
        uint256 key = input & 15;
        unchecked {
            for (uint256 i; i < 16; ++i) {
                value = uint256(keccak256(abi.encode(value, input, i))) & 255;
            }
            balances[key] += value;
        }
        if (input & 1 == 0) {
            try this.nested(input) {} catch {}
        }
        assert(value <= 255);
        return value;
    }

    function nested(uint256 input) external pure {
        require(input & 3 != 0, "nested revert");
    }

    function payload(bytes calldata data, string calldata label, address target)
        external returns (bytes memory, string memory)
    {
        value = uint256(keccak256(abi.encode(data, label, target))) & 255;
        assert(value <= 255);
        return (data, label);
    }

    function property_bounded() external view returns (bool) {
        return value <= 255;
    }

    function optimize_value() external view returns (int256) {
        return int256(value);
    }
}
