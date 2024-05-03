> **Definition**: Stateful fuzzing is the process of maintaining EVM state across multiple fuzzed transactions.

Stateful fuzzing is an incredibly powerful feature because it allows medusa to test your system **end-to-end**. Let's
take, for example, a staking system where you have the ability to `deposit`, `stake`, `unstake`, and `withdraw`. Because
medusa can execute an array of transactions, medusa can call [`deposit`, `stake`, `unstake`, `withdraw`] inorder and test the
whole system in one fell swoop. It is very important to note that medusa was not _forced_ to call those functions in
sequence. Medusa, over time, will identify that calling deposit allows it to stake tokens and having a staked balance
allows it to unstake, and so on.

In contrast, having a call sequence length of 1 is called **stateless fuzzing**.

> **Definition**: Stateless fuzzing is the process of executing a single transaction before resetting the EVM state.

Stateless fuzzing is useful for arithmetic libraries or isolated functions where state does not need to be maintained
across transactions. Stateless fuzzing, although faster, is not useful for larger systems that have many code paths with
nuanced and complex invariants.
