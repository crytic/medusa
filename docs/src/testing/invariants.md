# Types of Invariants

As discussed in the [testing overview](./overview.md) chapter, invariants describe the "truths" of your system. These
are unchanging properties that arise from the design of a codebase.

> **Note**: We will interchange the use of the word property and invariant often. For all intents and purposes, they
> mean the same thing.

Defining and testing your invariants is critical to assessing the **expected system behavior**.

We like to break down invariants into two general categories: function-level invariants and system-level invariants.
Note that there are other ways of defining and scoping invariants, but this distinction is generally sufficient to
start fuzz testing even the most complex systems.

## Function-level invariants

A function-level invariant can be defined as follows:

> **Definition**: A function-level invariant is a property that arises from the execution of a specific function.

Let's take the following function from a smart contract:

```solidity
function deposit() public payable {
    // Make sure that the total deposited amount does not exceed the limit
    uint256 amount = msg.value;
    require(totalDeposited + amount <= MAX_DEPOSIT_AMOUNT);

    // Update the user balance and total deposited
    balances[msg.sender] += amount;
    totalDeposited += amount;

    emit Deposit(msg.sender, amount, totalDeposited);
}
```

The `deposit` function has the following function-level invariants:

1. The ETH balance of `msg.sender` must decrease by `amount`.
2. The ETH of `address(this)` must increase by `amount`.
3. `balances[msg.sender]` should increase by `amount`.
4. The `totalDeposited` value should increase by `amount`.

Note that there other properties that can also be tested for but the above should highlight what a function-level
invariant is. In general, function-level invariants can be identified by assessing what must be true _before_ the execution
of a function and what must be true _after_ the execution of that same function. In the next chapter, we will write a
fuzz test to test the `deposit` function and how to use medusa to run that test.

Let's now look at system-level invariants.

## System-level invariants

A system-level invariant can be defined as follows:

> **Definition**: A system-level invariant is a property that holds true across the _entire_ execution of a system

Thus, a system-level invariant is a lot more generalized than a function-level invariant. Here are two common examples
of a function-level invariant:

1. The `xy=k` constant product formula should always hold for Uniswap pools
2. No user's balance should ever exceed the total supply for an ERC20 token.

In the `deposit` function above, we also see the presence of a system-level invariant:

**The `totalDeposited` amount should always be less than or equal to the `MAX_DEPOSIT_AMOUNT`**.

Since the `totalDeposited` value can be affected by the presence of other functions in the system
(e.g. `withdraw` or `stake`), it is best tested at the system level instead of the function level. We will look at how
to write system-level invariants in the [Writing System-Level Invariants](./writing-system-level-invariants.md) chapter.
