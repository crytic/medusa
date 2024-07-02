# Testing Overview

This chapter discusses the overarching goal of smart contract fuzzing.

Traditional fuzz testing (e.g. with [`AFL`](https://lcamtuf.coredump.cx/afl/)) aims to generally explore a binary by providing
random inputs in an effort to identify new system states or crash the program (please note that this is a pretty crude generalization).
This model, however, does not translate to the smart contract ecosystem since you cannot cause a smart contract to "crash".
A transaction that reverts, for example, is not equivalent to a binary crashing or panicking.

Thus, with smart contracts, we have to change the fuzzing paradigm. When you hear of "fuzzing smart contracts", you are
not trying to crash the program but, instead, you are trying to validate the **invariants** of the program.

> **Definition**: An invariant is a property that remains unchanged after one or more operations are applied to it.

More generally, an invariant is a "truth" about some system. For smart contracts, this can take many faces.

1. **Mathematical invariants**: `a + b = b + a`. The commutative property is an invariant and any Solidity math library
   should uphold this property.
2. **ERC20 tokens**: The sum of all user balances should never exceed the total supply of the token.
3. **Automated market maker (e.g. Uniswap)**: `xy = k`. The constant-product formula is an invariant that maintains the
   economic guarantees of AMMs such as Uniswap.

> **Definition**: Smart contract fuzzing uses random sequences of transactions to test the invariants of the smart contract system.

Before we explore how to identify, write, and test invariants, it is beneficial to understand how smart contract fuzzing
works under-the-hood.
