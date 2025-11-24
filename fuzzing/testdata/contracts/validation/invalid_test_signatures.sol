// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title InvalidTestSignatures
 * @notice Test contract to validate that medusa warns about incorrectly implemented test functions
 */
contract InvalidTestSignatures {
    // ===== VALID TESTS =====

    function property_validTest() public pure returns (bool) {
        return true;
    }

    function optimize_validTest() public pure returns (int256) {
        return 100;
    }

    // ===== INVALID PROPERTY TESTS =====

    // Invalid: wrong return type (returns uint256 instead of bool)
    function property_wrongReturnType() public pure returns (uint256) {
        return 1;
    }

    // Invalid: takes input parameter (should take no inputs)
    function property_hasInput(uint256 x) public pure returns (bool) {
        return x > 0;
    }

    // Invalid: no return value (should return bool)
    function property_noReturn() public pure {
        // Does nothing
    }

    // Invalid: multiple return values (should return only bool)
    function property_multipleReturns() public pure returns (bool, uint256) {
        return (true, 1);
    }

    // Invalid: returns wrong type with inputs
    function property_wrongTypeAndInput(address user) public pure returns (uint256) {
        return uint256(uint160(user));
    }

    // ===== INVALID OPTIMIZATION TESTS =====

    // Invalid: wrong return type (returns bool instead of int256)
    function optimize_wrongReturnType() public pure returns (bool) {
        return true;
    }

    // Invalid: takes input parameter (should take no inputs)
    function optimize_hasInput(uint256 x) public pure returns (int256) {
        return int256(x);
    }

    // Invalid: returns uint256 instead of int256
    function optimize_returnsUint() public pure returns (uint256) {
        return 100;
    }

    // Invalid: no return value (should return int256)
    function optimize_noReturn() public pure {
        // Does nothing
    }

    // Invalid: returns int128 instead of int256
    function optimize_wrongIntSize() public pure returns (int128) {
        return 100;
    }
}
