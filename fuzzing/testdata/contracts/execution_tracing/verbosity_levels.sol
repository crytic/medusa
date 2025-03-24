// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title Verbosity Levels Test Contract
 * @dev This contract is specifically designed to test Medusa's verbosity levels:
 * 
 * - Verbose (0): Only shows top-level calls and their events/return data
 *   (Events from HelperContract won't be visible, but events from TestContract will)
 * 
 * - VeryVerbose (1): Shows nested calls and their events/return data in last element
 *   (Shows TestContract -> HelperContract calls and all events in the failing transaction)
 * 
 * - VeryVeryVerbose (2): Shows all call sequences with complete event/return data
 *   (Shows setup calls like setXValue + the failing transaction with all details)
 * 
 * The test creates a multi-step sequence:
 * 1. Setup: Call setXValue which calls helper.setX
 * 2. Main call: setYValue which calls _processYValue
 * 3. Nested call: _processYValue calls helper.setY
 * 4. Assertion fails in _processYValue
 */

/**
 * @title HelperContract
 * @dev Secondary contract that's called by the main contract
 * Used to test nested call visibility in different verbosity levels
 */
contract HelperContract {
    uint256 xValue;
    uint256 yValue;
    
    // Events that will be emitted during calls
    // These should be visible in VeryVerbose and VeryVeryVerbose modes
    // but NOT in Verbose mode (since they're from nested calls)
    event setUpX(uint256 xValue);
    event setUpY(uint256 yValue);

    constructor() {
        xValue = 0;
        yValue = 0;
    }
    
    /**
     * @dev Sets the x value and emits an event
     * @param newValue The new value to set
     * @return true if successful
     */
    function setX(uint256 newValue) public returns (bool) {
        xValue = newValue;
        emit setUpX(xValue);
        return true;
    }
    
    /**
     * @dev Sets the y value and emits an event
     * @param newValue The new value to set
     * @return true if successful
     */
    function setY(uint256 newValue) public returns (bool) {
        yValue = newValue;
        emit setUpY(yValue);
        return true;
    }
}

/**
 * @title TestContract
 * @dev Main contract that creates a nested call structure
 * The pattern of calls is designed to test different verbosity levels
 */
contract TestContract {
    HelperContract public helper;
    bool isXValueSet;
    bool isYValueSet;

    event settingUpY(uint256 yValue);
    event setUpCompleted();

    constructor() {
        helper = new HelperContract();
    }

    /**
     * @dev Setup function that will be called first
     * @param value The x value to set
     */
    function setXValue(uint256 value) public {
        require(value > 0, "Value should be greater than 0");
        
        // Call the helper - nested call should only be visible in VeryVeryVerbose
        helper.setX(value);
        
        // Set flag to indicate successful setup
        isXValueSet = true;
    }

    /**
     * @dev Main function that triggers the failing transaction
     * @param value The y value to set
     * @return true if successful (never returns due to assertion failure)
     */
    function setYValue(uint256 value) public returns (bool) {
        // Check if setup was completed
        require(isXValueSet, "X Value is not set");
        emit settingUpY(value);

        _processYValue(value);
        emit setUpCompleted();
        isXValueSet = false;
        isYValueSet = false;
        return true;
    }
    
    /**
     * @dev Internal function that makes a nested call and then fails
     * @param yValue The value to process
     */
    function _processYValue(uint256 yValue) internal {
        require(yValue > 0, "Y value should be greater than zero");
        bool success = helper.setY(yValue);
        isYValueSet = true;

        // Assertion failure - tests trace at failure point
        assert(false);
    }
}
