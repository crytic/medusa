// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title Verbosity Levels Test Contract
 * @dev This file contains contracts specifically designed to test Medusa's verbosity levels:
 *
 * - Verbose (Level 0): Only shows top-level calls, hides nested calls
 * - VeryVerbose (Level 1): Shows nested calls with event and return data (default)
 * - VeryVeryVerbose (Level 2): Shows all call sequence elements with event and return data
 *
 * The test creates a complex call structure with nested function calls across multiple contracts:
 * 1. TestMarketplace.buyItem calls an internal function _processPayment
 * 2. _processPayment calls external functions on the TestToken contract:
 *    - balanceOf (view function)
 *    - allowance (view function)
 *    - transferFrom (state-changing function that emits an event)
 * 3. An assertion failure occurs in _processPayment (line 122)
 *
 * This structure allows us to verify that each verbosity level correctly:
 * - Shows/hides the appropriate level of call depth
 * - Includes/excludes internal function calls
 * - Shows traced events at different verbosity levels
 */

// CheatCodes interface for Medusa - similar to Foundry's VM
interface CheatCodes {
    function prank(address) external;
    function startPrank(address) external;
    function stopPrank() external;
    function deal(address who, uint256 newBalance) external;
    function warp(uint256) external;
    function roll(uint256) external;
}

/**
 * @title TestToken
 * @dev Simple ERC20 token contract that will be called by the TestMarketplace
 * This contract has both view and non-view functions with events to test
 * different verbosity levels of execution tracing
 */
contract TestToken {
    string public name;
    uint256 public totalSupply;
    mapping(address => uint256) private balances;
    mapping(address => mapping(address => uint256)) private allowances;

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);

    constructor(string memory _name, uint256 _initialSupply) {
        name = _name;
        totalSupply = _initialSupply;
        balances[msg.sender] = _initialSupply;
        emit Transfer(address(0), msg.sender, _initialSupply);
    }

    /**
     * @dev View function to check balance
     * This function will be called by the TestMarketplace
     * For verbosity testing, this tests if view function calls are properly shown/hidden
     */
    function balanceOf(address account) public view returns (uint256) {
        return balances[account];
    }

    /**
     * @dev Transfer function that emits an event
     * Not directly called in our test scenario, but illustrates a basic token transfer
     */
    function transfer(address to, uint256 amount) public returns (bool) {
        require(balances[msg.sender] >= amount, "Insufficient balance");
        balances[msg.sender] -= amount;
        balances[to] += amount;
        emit Transfer(msg.sender, to, amount);
        return true;
    }

    /**
     * @dev Approve function called as part of setup
     * Emits an Approval event which should be visible in higher verbosity levels
     */
    function approve(address spender, uint256 amount) public returns (bool) {
        allowances[msg.sender][spender] = amount;
        emit Approval(msg.sender, spender, amount);
        return true;
    }

    /**
     * @dev View function to check allowance
     * Called by TestMarketplace's _processPayment function
     */
    function allowance(address owner, address spender) public view returns (uint256) {
        return allowances[owner][spender];
    }

    /**
     * @dev TransferFrom function that will be called by TestMarketplace
     * This is a nested call that should be visible in VeryVerbose and VeryVeryVerbose modes
     * The event emitted here should also be visible in those modes
     */
    function transferFrom(address from, address to, uint256 amount) public returns (bool) {
        require(balances[from] >= amount, "Insufficient balance");
        require(allowances[from][msg.sender] >= amount, "Insufficient allowance");

        balances[from] -= amount;
        balances[to] += amount;
        allowances[from][msg.sender] -= amount;

        // This event should be visible in the trace at higher verbosity levels
        emit Transfer(from, to, amount);
        return true;
    }
}

/**
 * @title TestMarketplace
 * @dev Marketplace contract that creates a complex call structure with the TestToken
 * This contract is designed to test different verbosity levels in Medusa's execution tracing
 * by creating:
 * 1. Top-level calls (buyItem)
 * 2. Internal function calls (_processPayment)
 * 3. Cross-contract calls to TestToken (balanceOf, allowance, transferFrom)
 * 4. Event emissions at different levels
 * 5. An assertion failure deep in the call stack
 */
contract TestMarketplace {
    struct Listing {
        uint256 price;
        address seller;
        bool active;
    }

    mapping(uint256 => Listing) public listings;
    TestToken public paymentToken;

    // Events that will be emitted during testing
    // These should be visible at different verbosity levels
    event ItemListed(uint256 indexed itemId, address indexed seller, uint256 price);
    event ItemSold(uint256 indexed itemId, address indexed seller, address indexed buyer, uint256 price);
    event ItemDelisted(uint256 indexed itemId);

    constructor(address _paymentToken) {
        paymentToken = TestToken(_paymentToken);
    }

    /**
     * @dev Creates a listing that can be bought
     * This function will be part of the setup call sequence
     * In VeryVeryVerbose mode, this call and its emitted event should be traced
     */
    function listItem(uint256 itemId, uint256 price) public {
        require(price > 0, "Price must be positive");
        require(!listings[itemId].active, "Item already listed");

        listings[itemId] = Listing({
            price: price,
            seller: msg.sender,
            active: true
        });

        // This event should be visible in the trace at the highest verbosity level
        emit ItemListed(itemId, msg.sender, price);
    }

    /**
     * @dev Top-level function that triggers complex nested calls
     * This is the main entry point that Medusa will call during testing
     * It's a top-level call that should appear in ALL verbosity levels
     */
    function buyItem(uint256 itemId) public {
        Listing memory listing = listings[itemId];
        require(listing.active, "Item not listed");

        // Call an internal function to handle payment
        // This internal call tests if internal function calls are properly shown/hidden
        _processPayment(listing.seller, msg.sender, listing.price);

        // Mark item as sold
        listings[itemId].active = false;

        // This event should be visible in the trace in ALL verbosity levels
        emit ItemSold(itemId, listing.seller, msg.sender, listing.price);
    }

    /**
     * @dev Internal function that makes cross-contract calls and contains an assertion
     * This creates a multi-level call structure to test verbosity levels:
     * 1. Internal call from buyItem
     * 2. External calls to TestToken (balanceOf, allowance, transferFrom)
     * 3. Contains assertion failure for testing
     */
    function _processPayment(address seller, address buyer, uint256 amount) internal {
        // External call to view function - tests if nested view calls are traced
        uint256 buyerBalance = paymentToken.balanceOf(buyer);
        require(buyerBalance >= amount, "Insufficient balance");

        // Another external call to view function
        uint256 buyerAllowance = paymentToken.allowance(buyer, address(this));
        require(buyerAllowance >= amount, "Insufficient allowance");

        // External call that modifies state and emits an event
        bool success = paymentToken.transferFrom(buyer, seller, amount);
        
        // This assertion will fail, causing a test failure
        // It's located deep in the call stack to test if execution traces
        // properly show the failure location at different verbosity levels
        assert(false);
        
        require(success, "Transfer failed");
    }

    /**
     * @dev Function to delist an item
     * Not directly used in our verbosity tests
     */
    function delistItem(uint256 itemId) public {
        require(listings[itemId].active, "Item not listed");
        require(listings[itemId].seller == msg.sender, "Not the seller");

        listings[itemId].active = false;

        emit ItemDelisted(itemId);
    }
}