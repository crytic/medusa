# Writing Function-Level Invariants

This chapter will walk you through writing function-level fuzz tests for the `deposit` function that we saw in the 
[previous chapter](./invariants.md#function-level-invariants).

Before we write the fuzz tests, let's look into how we would write a unit test for the `deposit` function

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

What we will notice about the test above is that it _fixes_ the value that is being sent. It is unable to
test how the `deposit` function behaves across a variety of input spaces. Thus, a function-level fuzz test can be thought
of as a "unit test on steroids". Instead of fixing the `amount`, we let the fuzzer control the `amount` value to
any number between `[0, type(uint256).max`] and see how the system behaves to that.

> **Note**: One of the core differences between a traditional unit test versus a fuzz test is that a fuzz test accepts
> input arguments that the fuzzer can control.

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

Notice that we bounded the `_amount`  variable to be less than the amount of `usdc` tokens that `address(this)` has. This
type of bounding is very common when writing fuzz tests. Bounding allows you to only test values that are reasonable.
If `address(this)` doesn't have enough tokens, it does not make sense to try and call the `deposit` function. Additionally,
although we only tested one of the function-level invariants from the [previous chapter](./invariants.md), 
writing the remaining would follow the same pattern as the one written above.

## Running a function-level test with medusa

Let's now run the above example with medusa. Here is the test code:

```solidity
contract DepositContract {
    
    
}


TODO
```







