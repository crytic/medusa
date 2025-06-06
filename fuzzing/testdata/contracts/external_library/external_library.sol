// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title Library1
 * @dev A basic library that demonstrates a simple public function returning a string.
 */
library Library1 {
    /**
     * @dev Returns a static string "Library"
     * @return A static string value
     */
    function getLibrary() public pure returns(string memory) {
        return "Library";
    }
}

/**
 * @title Library2
 * @dev A library that depends on Library1, demonstrating direct library dependencies.
 * This library calls Library1's function and returns its result.
 */
library Library2 {
    /**
     * @dev Calls and returns the result from Library1.getLibrary()
     * @return A string value from the dependent library
     */
    function getLibrary() public pure returns(string memory) {
        return Library1.getLibrary();
    }
}

/**
 * @title Library3
 * @dev A library with multiple dependencies, demonstrating complex dependency chains.
 * This library calls both Library1 and Library2, creating a transitive dependency.
 */
library Library3 {
    /**
     * @dev Calls Library2.getLibrary() and then returns the result from Library1.getLibrary()
     * @return A string value from Library1
     */
    function getLibrary() public pure returns(string memory) {
        Library2.getLibrary();
        return Library1.getLibrary();
    }
}

/**
 * @title TestExternalLibrary
 * @dev A contract that uses the external libraries defined above.
 * This contract serves as a test case for the Medusa fuzzer to verify proper
 * library resolution, deployment ordering, and ABI handling.
 */
contract TestExternalLibrary {
    /**
     * @dev A function designed to be targeted by the fuzzer
     * Contains a deliberate assertion failure when Library3 returns "Library"
     * @return false this line never gets executed
     * @notice This function is used to test if the fuzzer can properly resolve
     * library dependencies
     */
    function fuzz_me() public returns(bool){
       if (keccak256(abi.encodePacked(Library3.getLibrary())) == keccak256(abi.encodePacked("Library"))) {
            assert(false);
            return false;
       }
    }
}