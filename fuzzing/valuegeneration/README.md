# Value generation
This directory is used for generating and mutating values in call sequences for fuzzing. We have two interfaces: `ValueGenerator` for generating random values from scratch (methods include `GenerateAddress() common.Address`), and `ValueMutator` for randomly mutating existing values (methods include `MutateAddress(addr common.Address) common.Address`).
We have `RandomValueGenerator` for generating new random values; it implements `ValueGenerator`.
We have `MutationalValueGenerator` for mutating values during fuzzing and `ShrinkingValueMutator` for attempting to shrink values when shrinking a failed test case during fuzzing. Both implement `ValueMutator`. `MutationalValueGenerator` sometimes generates completely new random values instead of mutating existing ones, as determined by its configuration. It uses a `RandomValueGenerator` to do this.
Currently `RandomValueGenerator` implements `ValueMutator` and `MutationalValueGenerator` implements `ValueGenerator`. We are planning to remove these implementations eventually; see issue #809.
`ValueSet` represents potentially-important constants taken from solidity files. It can be gathered from an AST (`value_set_from_ast.go`) or from Slither (`value_set_from_slither.go`).
`abi_values.go` contains wrappers around our mutators and generators: `func GenerateAbiValue(generator ValueGenerator, inputType *abi.Type) any`, `func MutateAbiValue(generator ValueGenerator, mutator ValueMutator, inputType *abi.Type, value any)`. It also contains various functions for converting values back and forth between ABI, JSON, and string. We will likely move these somewhere else; see issue #809.

(TODO: Should I make this more high-level? Should I go into more detail about how values are generated?)
