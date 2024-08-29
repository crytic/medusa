# Coverage Reports

This chapter will walk you through how to maximize coverage reports for your tests.

## What is a coverage report

A coverage report is a document or summary that provides information about how much of a codebase is covered by tests. It measures the extent to which the code is executed when the test suite runs, typically expressed as a percentage. The goal of a coverage report is to give developers insights into how thoroughly the code is tested, helping identify areas that might need additional testing to ensure reliability and reduce bugs.

## Setting up coverage report for medusa

In order to get a coverage report when running medusa tests, you need to set the corpusDirectory in `medusa`'s project configuration. After successfully running a test, the corpusDirectory will contain the coverage_report.html and other folders like call_sequences and tests_results.

## Running a coverage test with medusa

Let's now run a coverage report with medusa. Here is the test code:

```solidity
contract DepositContract {
    // @notice MAX_DEPOSIT_AMOUNT is the maximum amount that can be deposited into this contract
    uint256 public constant MAX_DEPOSIT_AMOUNT = 1_000_000e18;
    uint256 public constant MIN_DEPOSIT_AMOUNT = 1e18;

    // @notice balances holds user balances
    mapping(address => uint256) public balances;

    // @notice totalDeposited represents the current deposited amount across all users
    uint256 public totalDeposited;

    // @notice Deposit event is emitted after a deposit occurs
    event Deposit(address depositor, uint256 amount, uint256 totalDeposited);

    // @notice Withdraw event is emitted after a withdraw occurs
    event Withdraw(address user, uint256 amount);

    // @notice deposit allows user to deposit into the system
    function deposit() public payable {
        // Make sure that the total deposited amount does not exceed the limit
        uint256 amount = msg.value;
        require(totalDeposited + amount <= MAX_DEPOSIT_AMOUNT);

        // Update the user balance and total deposited
        balances[msg.sender] += amount;
        totalDeposited += amount;

        emit Deposit(msg.sender, amount, totalDeposited);
    }

    function withdraw(uint256 amount, address _addr) public {
        require(balances[msg.sender] > amount, "insufficient balance");
        balances[msg.sender] -= amount;
        totalDeposited -= amount;
        (bool sent,) = _addr.call{value: amount}("");
        require(sent, "Failed to send Ether");

        emit Withdraw(msg.sender, amount);
    }
}
```

```solidity
contract TestDeposit {
    DepositContract public depositContract;

    constructor() {
        depositContract = new DepositContract();
    }

    function testDeposit(uint256 amount) public {
        // Retrieve balance of user before deposit
        uint256 preBalance = depositContract.balances(address(this));

        depositContract.deposit{value: amount}();

        // Retrieve balance of user after deposit
        uint256 afterBalance = depositContract.balances(address(this));

        // Assertion
        assert(preBalance - amount == afterBalance);
    }
}
```

NB: The coverage report is gotten when we stop running our medusa fuzz command.

![alt text](https://res.cloudinary.com/josh4324/image/upload/v1724324363/Screenshot_2024-08-22_at_11.59.10_syoulj.png "Coverage Report")

The sections of the code highlighted with the success color indicate areas that have been tested, while those highlighted with the error color indicate areas of the code that have not been tested.

The √ indicates that thesource line executed without reverting.

The ⟳ indicated the the source line executed, but was reverted.

The coverage report shows the no of files tested, the number of lines covered and the percentage of the codebase that was tested.
