## Writing System-Level Invariants

This chapter will walk you through writing system-level fuzz tests. We will continue using the `DepositContract` from the [previous chapter](./writing-function-level-invariants.md) and extend it with a `withdraw` function to demonstrate why system-level invariants require testing across arbitrary sequences of transactions.

Before we begin, let's recall the definition from the [Types of Invariants](./invariants.md#system-level-invariants)
chapter:

> **Definition**: A system-level invariant is a property that holds true across the _entire_ execution of a system.

Unlike function-level invariants that assert pre/post-conditions of a _single_ function call, system-level invariants must hold **regardless of the order or combination of function calls** made to the system.

### Identifying system-level invariants

Let's extend our `DepositContract` with a `withdraw` function:

```solidity
contract DepositContract {
    uint256 public constant MAX_DEPOSIT_AMOUNT = 1_000_000e18;

    mapping(address => uint256) public balances;
    uint256 public totalDeposited;

    event Deposit(address depositor, uint256 amount, uint256 totalDeposited);
    event Withdrawal(address withdrawer, uint256 amount, uint256 totalDeposited);

    function deposit() public payable {
        uint256 amount = msg.value;
        require(totalDeposited + amount <= MAX_DEPOSIT_AMOUNT);

        balances[msg.sender] += amount;
        totalDeposited += amount;

        emit Deposit(msg.sender, amount, totalDeposited);
    }

    function withdraw(uint256 amount) public {
        require(balances[msg.sender] >= amount);

        balances[msg.sender] -= amount;
        totalDeposited -= amount;

        payable(msg.sender).transfer(amount);

        emit Withdrawal(msg.sender, amount, totalDeposited);
    }
}
```

Now that multiple functions can modify `totalDeposited` and user balances, we can identify system-level invariants that must hold after _any_ sequence of deposits and withdrawals:

1. `totalDeposited` should always be less than or equal to `MAX_DEPOSIT_AMOUNT`.
2. `totalDeposited` should always equal the ETH balance of the contract (`address(this).balance`).
3. No individual user's balance should ever exceed `totalDeposited`.

These properties cannot be fully validated by testing a single function in isolation. For example, a bug might only manifest after a specific sequence of deposits and withdrawals. This is exactly where system-level invariant testing shines.

### Writing property tests

System-level invariants are tested using **property tests**. Property tests in Medusa are Solidity functions that:

1. Are prefixed with `property_` (configurable via `testing.propertyTesting.testPrefixes`).
2. Accept **no input arguments**.
3. Return a `bool` (`true` = passing, `false` = failing).

> **Note**: Property tests should be `view` or `pure` functions since they are meant to observe the system state, not modify it.

Medusa calls every property test **after each transaction** in a call sequence. If any property test returns `false`, Medusa reports a failure along with the call sequence that caused it. This means that your property tests act as continuous monitors of the system's health throughout the entire fuzzing campaign.

Here is what our property tests look like:

```solidity
contract TestDepositContract {
    DepositContract depositContract;

    constructor() payable {
        depositContract = new DepositContract();
    }

    // Allow this contract to receive ETH from withdrawals
    receive() external payable {}

    // @notice the totalDeposited amount should never exceed the MAX_DEPOSIT_AMOUNT
    function property_total_deposited_lte_max() public view returns (bool) {
        return depositContract.totalDeposited() <= depositContract.MAX_DEPOSIT_AMOUNT();
    }

    // @notice the totalDeposited amount should always equal the contract's ETH balance
    function property_total_deposited_eq_balance() public view returns (bool) {
        return depositContract.totalDeposited() == address(depositContract).balance;
    }

    // @notice no user's balance should ever exceed the totalDeposited amount
    function property_user_balance_lte_total() public view returns (bool) {
        return depositContract.balances(address(this)) <= depositContract.totalDeposited();
    }
}
```

Notice the key differences from the function-level fuzz test in the [previous chapter](./writing-function-level-invariants.md):

- **No input arguments**: The fuzzer does _not_ control the inputs to property tests. Instead, the fuzzer generates
  random call sequences targeting all public functions (e.g. `deposit`, `withdraw`) and the property tests passively
  observe whether the resulting state is valid.
- **No direct function calls**: Property tests do not call into the system under test. They only _read_ state and
  return whether the invariant holds.
- **Multiple invariants**: Each property test checks a single invariant. This makes it easy to identify exactly which
  invariant was violated when a test fails.

### Comparing to function-level invariants

It is worth understanding the difference between how function-level and system-level invariants are tested:

|                           | Function-level (assertion tests)                                                            | System-level (property tests)                              |
| ------------------------- | ------------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| **Where invariants live** | Inside the function being tested (via `assert()`) or in a fuzz test that calls the function | In standalone `property_` functions                        |
| **Inputs**                | Fuzzer controls function arguments                                                          | Fuzzer controls the _call sequence_, not the property test |
| **When checked**          | During execution of the function                                                            | After _every_ transaction in a call sequence               |
| **What they test**        | Pre/post-conditions of a single function                                                    | Global properties across arbitrary sequences of calls      |
| **Call sequence length**  | Typically `1` (stateless)                                                                   | Typically `> 1` (stateful)                                 |

### Running a system-level test with Medusa

Here is the complete test contract:

```solidity
contract DepositContract {
    uint256 public constant MAX_DEPOSIT_AMOUNT = 1_000_000e18;

    mapping(address => uint256) public balances;
    uint256 public totalDeposited;

    event Deposit(address depositor, uint256 amount, uint256 totalDeposited);
    event Withdrawal(address withdrawer, uint256 amount, uint256 totalDeposited);

    function deposit() public payable {
        uint256 amount = msg.value;
        require(totalDeposited + amount <= MAX_DEPOSIT_AMOUNT);

        balances[msg.sender] += amount;
        totalDeposited += amount;

        emit Deposit(msg.sender, amount, totalDeposited);
    }

    function withdraw(uint256 amount) public {
        require(balances[msg.sender] >= amount);

        balances[msg.sender] -= amount;
        totalDeposited -= amount;

        payable(msg.sender).transfer(amount);

        emit Withdrawal(msg.sender, amount, totalDeposited);
    }
}

contract TestDepositContract {
    DepositContract depositContract;

    constructor() payable {
        depositContract = new DepositContract();
    }

    // Allow this contract to receive ETH from withdrawals
    receive() external payable {}

    // @notice wrapper for deposit that the fuzzer can call
    function deposit(uint256 _amount) public {
        uint256 amount = clampLte(_amount, address(this).balance);
        depositContract.deposit{value: amount}();
    }

    // @notice wrapper for withdraw that the fuzzer can call
    function withdraw(uint256 _amount) public {
        uint256 amount = clampLte(_amount, depositContract.balances(address(this)));
        depositContract.withdraw(amount);
    }

    // @notice the totalDeposited amount should never exceed the MAX_DEPOSIT_AMOUNT
    function property_total_deposited_lte_max() public view returns (bool) {
        return depositContract.totalDeposited() <= depositContract.MAX_DEPOSIT_AMOUNT();
    }

    // @notice the totalDeposited amount should always equal the contract's ETH balance
    function property_total_deposited_eq_balance() public view returns (bool) {
        return depositContract.totalDeposited() == address(depositContract).balance;
    }

    // @notice no user's balance should ever exceed the totalDeposited amount
    function property_user_balance_lte_total() public view returns (bool) {
        return depositContract.balances(address(this)) <= depositContract.totalDeposited();
    }

    // @notice clampLte returns a value between [0, b]
    function clampLte(uint256 a, uint256 b) internal pure returns (uint256) {
        if (!(a <= b)) {
            return a % (b + 1);
        }
        return a;
    }
}
```

Note that we added `deposit` and `withdraw` wrapper functions in `TestDepositContract`. These wrappers allow the fuzzer to call `deposit` and `withdraw` with fuzzed arguments while automatically bounding the inputs to valid ranges.
The property test functions then check the invariants after each of these calls. Because the wrappers call `DepositContract`
internally, `msg.sender` inside `DepositContract` is always `TestDepositContract` in this example. This keeps the example
focused on stateful call sequences rather than multi-user behavior.

To run this test contract, download the project configuration file [here](../static/system_level_testing_medusa.json), rename it to `medusa.json`, and run:

```
medusa fuzz --config medusa.json
```

The following non-default changes were made to the default project configuration file to allow this test to run:

- `fuzzing.targetContracts`: Updated to `["TestDepositContract"]`.
- `fuzzing.targetContractsBalances`: Updated to `["21267647932558653966460912964485513215"]` to give the test contract an ETH balance for depositing.
- `fuzzing.testLimit`: Set to `10_000` to run a reasonable campaign.

> **Note**: The default `callSequenceLength` of `100` is already well-suited for system-level testing. Unlike
> function-level testing where `callSequenceLength` is typically `1`, system-level testing benefits from longer call
> sequences. A longer sequence gives the fuzzer more room to explore complex state transitions that arise from
> interleaving multiple function calls.

### When a property test fails

If a property test fails, Medusa will halt and report the failing sequence. The output format looks like this
(illustrative example):

```
[FAILED] Property Test: TestDepositContract.property_total_deposited_eq_balance()
Test for method "TestDepositContract.property_total_deposited_eq_balance()" failed after the following call sequence:
[Call Sequence]
1) TestDepositContract.deposit([55408297438917960862474451975836818265]) (block=1, time=2, gas=12500000, gasprice=1, value=0, sender=0x10000)
2) TestDepositContract.withdraw([19028842074912045090817234023499421958]) (block=3, time=5, gas=12500000, gasprice=1, value=0, sender=0x20000)
3) TestDepositContract.deposit([7702981288038713856009128004564213903]) (block=4, time=8, gas=12500000, gasprice=1, value=0, sender=0x10000)
```

The exact sender addresses, calldata, and sequence length will depend on your configuration and the bug being
exercised. Medusa will also attempt to **shrink** the failing call sequence to the minimal set of transactions required
to reproduce the failure. This makes it significantly easier to diagnose the root cause of the invariant violation.

### Tips for writing effective system-level invariants

1. **Start with the specification**: Identify what must _always_ be true about your system regardless of usage patterns.
   Common examples include conservation of value, supply invariants, and access control guarantees.
2. **One invariant per property test**: Keep each `property_` function focused on a single invariant. This makes failures easier to diagnose.
3. **Use wrapper functions**: Create wrapper functions in your test contract that bound fuzzed inputs to valid ranges.
   This prevents the fuzzer from wasting time on transactions that will simply revert.
4. **Increase the call sequence length**: System-level bugs often require multiple transactions to surface. Use a `callSequenceLength` of `50` or more to give the fuzzer room to explore complex state transitions.
5. **Use multiple sender addresses when caller identity matters**: If your system under test depends on `msg.sender`,
   configure `fuzzing.senderAddresses` accordingly and ensure your harness preserves the original caller rather than
   routing every interaction through a single wrapper contract.
6. **Combine with assertion tests**: Property tests and assertion tests complement each other. Use assertion tests for function-level pre/post-conditions and property tests for system-wide invariants. Both can run simultaneously.
