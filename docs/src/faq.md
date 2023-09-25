# Frequently Asked Questions

**Why create a new fuzzer if Echidna is already a great fuzzer?**

With medusa, we are exploring a different EVM implementation and language for our smart contract fuzzer. While Echidna is already doing an amazing job, medusa offers the following advantages:

- It is written in Go, easing the maintenance and allowing the creation of a native API for future integration into other projects.
- It uses geth as a base, ensuring the EVM equivalence.

**Should I switch to medusa right away?**

We do not recommend switching to medusa until it is extensively tested. However we encourage you to try it, and [let us know your experience](https://github.com/trailofbits/medusa/issues). In that sense, Echidna is our robust and well tested fuzzer, while medusa is our new exploratory fuzzer. [Follow us](https://twitter.com/trailofbits/) to hear updates about medusa as it grows in maturity.

**Will all the previous available documentation from [secure-contracts.com](https://secure-contracts.com/) will apply to medusa?**

In general, yes. All the information on testing approaches and techniques will apply for medusa. There are, however, different configuration options names and a few missing or different features in medusa from Echidna that we will be updating over time.
