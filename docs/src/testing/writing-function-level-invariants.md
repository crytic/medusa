## Writing Function-Level Invariants


This chapter will walk you through writing function-level fuzz tests for the `deposit` function that we saw in the [previous chapter](./invariants.md#function-level-invariants).


Before we write the fuzz tests, let's look into how we would write a unit test for the `deposit` function:


```solidity
function testDeposit() public {
    // The amount of tokens to deposit
    uint256 amount = 10 ether;
    
    // Register the user
    depositContract.registerUser(address(this));
    
    // Retrieve balance of user before deposit
    preBalance = depositContract.balances(address(this));
    
    // Call the deposit contract (let's assume this contract has 10e18 tokens)
    depositContract.deposit(amount);
    
    // Assert post-conditions
    assert(depositContract.balances(msg.sender) == preBalance + amount);
    // Add other assertions here
}
```

What we will notice about the test above is that it _fixes_ the value that is being sent. It is unable to test how the `deposit` function behaves across a variety of input spaces. Thus, a function-level fuzz test can be thought of as a "unit test on steroids". Instead of fixing the `amount`, we let the fuzzer control the `amount` value to any number between `[0, type(uint256).max]` and see how the system behaves to that.

> **Note**: One of the core differences between a traditional unit test versus a fuzz test is that a fuzz test accepts input arguments that the fuzzer can control.

### Writing a Fuzz Test for the `deposit` Function

Here is what a fuzz test for the `deposit` function would look like:

```solidity
function testDeposit(uint256 _amount) public {
    // Let's bound the input to be _at most_ the amount of USDC tokens that this contract has
    // The amount value will now in between [0, usdc.balanceOf(address(this))]
    uint256 amount = clampLt(_amount, usdc.balanceOf(address(this)));
    
    // Register the user
    depositContract.registerUser(address(this));
    
    // Retrieve balance of user before deposit
    preBalance = depositContract.balances(address(this));
    
    // Call the deposit contract with a variable amount
    depositContract.deposit(amount);
    
    // Assert post-conditions
    assert(depositContract.balances(msg.sender) == preBalance + amount);
    // Add other assertions here
}
```

Notice that we bounded the `_amount` variable to be less than the amount of `usdc` tokens that `address(this)` has. This type of bounding is very common when writing fuzz tests. Bounding allows you to only test values that are reasonable. If `address(this)` doesn't have enough tokens, it does not make sense to try and call the `deposit` function. Additionally, although we only tested one of the function-level invariants from the [previous chapter](./invariants.md), writing the remaining would follow a similar pattern as the one written above.

## Running a function-level test with medusa

Let's now run the above example with medusa. Here is the test code:

```solidity
contract DepositContract {
    function deposit(uint256 amount) payable {
    }

}
contract TestDeposit is Test {
    DepositContract depositContract = new DepositContract();
    function testDeposit(uint256 _amount) public {
    // Let's bound the input to be _at most_ the amount of USDC tokens that this contract has
    // The amount value will now in between [0, usdc.balanceOf(address(this))]
    uint256 amount = clampLt(_amount, usdc.balanceOf(address(this)));
    // Register the user
    depositContract.registerUser(address(this));
    // Retrieve balance of user before deposit
    preBalance = depositContract.balances(address(this));
    // Call the deposit contract with a variable amount
    depositContract.deposit{value: amount}(amount);
    // Assert post-conditions
    assert(depositContract.balances(msg.sender) == preBalance + amount);
    // Add other assertions here
    }
}
```
o run the fuzz test, you can use the following command:

```
medusa fuzz --config medusa.yaml
```

Where `medusa.yaml` is a configuration file that specifies the settings for the fuzzing campaign.

### Additional Tips

- **Use a variety of input values.** The more input values you test, the more likely you are to find bugs.
- **Use a variety of input types.** Don't just test with integers. Try testing with strings, arrays, and other complex data types.
- **Use a variety of fuzzing techniques.** There are many different fuzzing techniques available. Experiment with different techniques to see which ones are most effective for your project.
- **Use a coverage-guided fuzzer.** A coverage-guided fuzzer will focus on testing the parts of your code that are most likely to contain bugs.
- **Use a symbolic execution engine.** A symbolic execution engine can help you to find bugs that are difficult to find with traditional fuzzing techniques.






