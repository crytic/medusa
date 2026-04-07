# Fuzzing Pre-Deployed Contracts (Genesis State)

By default, Medusa deploys your contracts from scratch at the start of every fuzzing campaign. This works
well for most projects, but some systems are difficult or impossible to reproduce through a simple deploy
script — for example, contracts deployed by a factory, contracts with complex constructor side-effects, or
multi-step deployment pipelines that use Foundry broadcast scripts.

The **genesis state** feature lets you snapshot an existing EVM state and hand it to Medusa as the
starting point for fuzzing. Medusa loads the snapshot, skips deploying the contracts that are already
present, and starts fuzzing immediately from the captured state — including all storage slots, balances,
and nonces.

## Workflow

### 1. Deploy your contracts with Foundry

Run your normal deployment script against a local Anvil node:

```bash
anvil &
forge script script/Deploy.s.sol --broadcast --rpc-url http://localhost:8545
```

Note the addresses that are printed after deployment — you will need them in step 3.

### 2. Capture the EVM state

```bash
cast rpc anvil_dumpState > state.json
```

This writes a single-line JSON file containing a gzip-compressed hex string of the full EVM state.
Medusa accepts this format directly without any preprocessing.

### 3. Configure Medusa

Add two fields to the `chainConfig` section of your `medusa.json`:

```json
{
  "chainConfig": {
    "genesisStateFile": "state.json",
    "genesisContractMappings": {
      "0x5FbDB2315678afecb367f032d93F642f64180aa3": "Counter",
      "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512": "MyToken"
    }
  }
}
```

- **`genesisStateFile`** — path to the snapshot file (relative to `medusa.json`).
- **`genesisContractMappings`** — maps each pre-deployed address to the contract name in your
  compilation artifacts. Medusa uses this to bind the correct ABI to the address.

**`genesisContractMappings` controls ABI binding only.** It does not control which contracts the
fuzzer generates calls to. The two concepts are separate:

| Goal                                                                               | Where to configure        |
| ---------------------------------------------------------------------------------- | ------------------------- |
| Make a genesis contract's functions callable (by the fuzzer or by other contracts) | `genesisContractMappings` |
| Have the fuzzer generate top-level calls to a contract and check its invariants    | `targetContracts`         |

Contracts you want to **fuzz** must appear in both. Contracts that are **dependencies** of your
targets (tokens, oracles, registries, etc.) should appear only in `genesisContractMappings` — they
need an ABI so internal calls resolve, but you do not want the fuzzer flooding them with calls.

```json
{
  "chainConfig": {
    "genesisStateFile": "state.json",
    "genesisContractMappings": {
      "0x5FbDB2315678afecb367f032d93F642f64180aa3": "Counter",
      "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512": "MyToken",
      "0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9": "USDC"
    }
  },
  "fuzzing": {
    "targetContracts": ["Counter", "MyToken"]
  }
}
```

In the example above, `USDC` is present on-chain and its ABI is registered (so `Counter` and
`MyToken` can call into it), but the fuzzer will not generate calls to `USDC` directly. You could
add `"USDC"` to `targetContracts` if you wanted the fuzzer to call its functions too — though for
most dependency contracts that is not what you want.

### 4. Run Medusa

```bash
medusa fuzz
```

On startup you will see log lines confirming that the state was loaded and the contracts were mapped:

```
⇾  Loaded genesis state from state.json with 12 accounts
⇾  Mapped genesis contract at 0x5FbDB...aa3 to Counter
⇾  Mapped genesis contract at 0xe7f1...512 to MyToken
```

Medusa will then fuzz the pre-deployed contracts exactly as if it had deployed them itself.

## Accepted File Formats

Medusa auto-detects the format — no configuration flag is required.

| Format                  | Description                                                                                                |
| ----------------------- | ---------------------------------------------------------------------------------------------------------- |
| **Native anvil dump**   | Output of `cast rpc anvil_dumpState`: a JSON-quoted gzip-compressed hex string starting with `"0x1f8b..."` |
| **Anvil wrapper JSON**  | Decompressed dump with a top-level `"block"` and `"accounts"` field                                        |
| **Plain accounts JSON** | A flat map of `"0xADDR"` → `{balance, nonce, code, storage}`                                               |

## How Genesis Accounts and Fuzzer Accounts Interact

Medusa always adds its own sender and deployer accounts on top of the loaded state. If a loaded address
collides with a Medusa-managed address, the Medusa account takes precedence. In practice this is unlikely
because Medusa's default addresses (`0x10000`, `0x20000`, `0x30000`) do not overlap with addresses
produced by Foundry's deterministic deployment.

## Limitations

- Contracts that are present in the genesis state but not listed in `genesisContractMappings` will be
  present on-chain but Medusa will not call them (it has no ABI for them).
- Library contracts do not need to be listed in `genesisContractMappings` or `targetContracts`; they are
  called indirectly through the contracts that use them.
- The genesis state is loaded once at startup. It is not re-loaded between worker resets — each worker
  always starts from the same captured state.
