## Writing Function-Level Invariants

This chapter will walk you through writing function-level fuzz tests for the `deposit` function that we saw in the [previous chapter](./invariants.md#function-level-invariants).

Before we write the fuzz tests, let's look into how we would write a unit test for the `deposit` function:

```solidity
function testDeposit() public {
    // The amount of tokens to deposit
    uint256 amount = 10 ether;

    // Retrieve balance of user before deposit
    preBalance = depositContract.balances(address(this));

    // Call the deposit contract (let's assume this contract has 10 ether)
    depositContract.deposit{value: amount}();

    // Assert post-conditions
    assert(depositContract.balances(msg.sender) == preBalance + amount);
    // Add other assertions here
}
```

What we will notice about the test above is that it _fixes_ the value that is being sent. It is unable to test how the
`deposit` function behaves across a variety of input spaces. Thus, a function-level fuzz test can be thought of as a
"unit test on steroids". Instead of fixing the `amount`, we let the fuzzer control the `amount` value to any number between
`[0, type(uint256).max]` and see how the system behaves to that.

> **Note**: One of the core differences between a traditional unit test versus a fuzz test is that a fuzz test accepts input arguments that the fuzzer can control.

### Writing a Fuzz Test for the `deposit` Function

Here is what a fuzz test for the `deposit` function would look like:

```solidity
function testDeposit(uint256 _amount) public {
    // Let's bound the input to be _at most_ the ETH balance of this contract
    // The amount value will now in between [0, address(this).balance]
    uint256 amount = clampLte(_amount, address(this).balance);

    // Retrieve balance of user before deposit
    uint256 preBalance = depositContract.balances(address(this));

    // Call the deposit contract with a variable amount
    depositContract.deposit{value: _amount}();

    // Assert post-conditions
    assert(depositContract.balances(address(this)) == preBalance + amount);
    // Add other assertions here
}
```

Notice that we bounded the `_amount` variable to be less than or equal to the test contract's ETH balance.
This type of bounding is very common when writing fuzz tests. Bounding allows you to only test values that are reasonable.
If `address(this)` doesn't have enough ETH, it does not make sense to try and call the `deposit` function. Additionally,
although we only tested one of the function-level invariants from the [previous chapter](./invariants.md), writing the remaining
would follow a similar pattern as the one written above.

## Running a function-level test with medusa

Let's now run the above example with medusa. Here is the test code:

```solidity
contract DepositContract {
    // @notice MAX_DEPOSIT_AMOUNT is the maximum amount that can be deposited into this contract
    uint256 public constant MAX_DEPOSIT_AMOUNT = 1_000_000e18;

    // @notice balances holds user balances
    mapping(address => uint256) public balances;

    // @notice totalDeposited represents the current deposited amount across all users
    uint256 public totalDeposited;

    // @notice Deposit event is emitted after a deposit occurs
    event Deposit(address depositor, uint256 amount, uint256 totalDeposited);

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
}

contract TestDepositContract {

    // @notice depositContract is an instance of DepositContract
    DepositContract depositContract;

    constructor() payable {
        // Deploy the deposit contract
        depositContract = new DepositContract();
    }

    // @notice testDeposit tests the DepositContract.deposit function
    function testDeposit(uint256 _amount) public {
        // Let's bound the input to be _at most_ the ETH balance of this contract
        // The amount value will now in between [0, address(this).balance]
        uint256 amount = clampLte(_amount, address(this).balance);

        // Retrieve balance of user before deposit
        uint256 preBalance = depositContract.balances(address(this));

        // Call the deposit contract with a variable amount
        depositContract.deposit{value: _amount}();

        // Assert post-conditions
        assert(depositContract.balances(address(this)) == preBalance + amount);
        // Add other assertions here
    }

    // @notice clampLte returns a value between [a, b]
    function clampLte(uint256 a, uint256 b) internal returns (uint256) {
        if (!(a <= b)) {
            uint256 value = a % (b + 1);
            return value;
        }
        return a;
    }

}
```

To run this test contract, download the project configuration file [here](../static/function_level_testing_medusa.json),
rename it to `medusa.json`, and run:

```
medusa fuzz --config medusa.json
```

The following changes were made to the default project configuration file to allow this test to run:

- `fuzzing.targetContracts`: The `fuzzing.targetContracts` value was updated to `["TestDepositContract"]`.
- `fuzzing.targetContractsBalances`: The `fuzzing.targetContractsBalances` was updated to `["0xfffffffffffffffffffffffffffffff"]`
  to allow the `TestDepositContract` contract to have an ETH balance allowing the fuzzer to correctly deposit funds into the
  `DepositContract`.
- `fuzzing.testLimit`: The `fuzzing.testLimit` was set to `1_000` to shorten the duration of the fuzzing campign.
- `fuzzing.callSequenceLength`: The `fuzzing.callSequenceLength` was set to `1` so that the `TestDepositContract` can be
  reset with its full ETH balance after each transaction.
