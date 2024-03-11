## Writing and Testing System-Level Invariants with Medusa

### Introduction

System-level invariants are properties that must remain true throughout the execution of a system. This means that no point in time should these properties be broken. Before writing fuzz tests, these properties should first be identified and better still, written in plain text. Only after that do we start writing the fuzz tests.

### Writing System-Level Invariants as Property Tests

Property tests are represented as functions within a Solidity contract whose names are prefixed with a prefix specified by the `testPrefixes` configuration option (`fuzz_` is the default test prefix). Additionally, they must take no arguments and return a `bool` indicating if the test succeeded.

For example, the following property test checks that no user's token balance exceeds the total number of tokens in the contract:

```solidity
contract TestContract is Token {
  function fuzz_userBalanaceShouldNotExceedTotalSupply() public returns (bool) {
    return balances[msg.sender] <= totalSupply;
  }
}
```

### Writing System-Level Invariants as Assertion Tests

Assertion tests check to see if a given call sequence can cause the Ethereum Virtual Machine (EVM) to "panic". The EVM has a variety of panic codes for different scenarios. For example, there is a unique panic code when an `assert(x)` statement returns `false` or when a division by zero is encountered.

For example, the following assertion test checks that the token's decimals do not change during fuzzing:

```solidity
contract TestContract {
  Token token;
  constructor() {
    token = new Token("MyToken", "MT", 18);
  }
  function testTokenDecimalsDoesNotChange() public {
    assert(token.decimals() == 18);
  }
}
```

### Testing System-Level Invariants with Medusa

TODO