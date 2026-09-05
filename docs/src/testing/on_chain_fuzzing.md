# On-Chain Fuzzing (Fork Mode)

By default, Medusa fuzzes contracts against a blockchain it creates from scratch: accounts are
provisioned, contracts are deployed at their deterministic addresses, and storage only ever contains
what the fuzzing campaign itself produced. This is ideal for testing your system in isolation, but
many bugs only appear when a system interacts with the environment it will actually live in — real
token balances, existing liquidity positions, live oracles, and governance state that took months to
accumulate.

**Fork mode** (also called on-chain fuzzing) changes where the blockchain's initial state comes from.
Instead of starting from an empty world, Medusa seeds its simulated chain with the state of a real
network at a block number you choose. When execution touches an account or storage slot that was not
created locally, the state is fetched transparently from the remote RPC endpoint and cached locally.
Your contracts are still deployed by Medusa on top of this forked state, so you can fuzz your system
as it would behave in production — against the real protocol it composes with.

> 🚩 Fork mode sends queries to a remote RPC endpoint. Make sure your RPC provider's terms of service
> allow the query volume, and see [The RPC Cache](#the-rpc-cache) for how Medusa keeps this volume low.

## Fork Mode vs. Genesis State

Fork mode is one of two ways to fuzz against state that Medusa did not create itself. The other is
[genesis state](genesis_state.md), which loads a captured snapshot file at startup. The features are
complementary and solve different problems:

|                                       | Fork mode                              | Genesis state                            |
| ------------------------------------- | -------------------------------------- | ---------------------------------------- |
| State source                          | Live RPC queries at a pinned block     | A snapshot file dumped from a local node |
| Network access during fuzzing         | Yes (lazy fetches, cached)             | None                                     |
| Works with public RPC endpoints       | Yes                                    | N/A (you own the source node)            |
| ABI binding to pre-existing contracts | Via interfaces in your test contracts  | Via `genesisContractMappings`            |
| Typical use case                      | Fuzzing against mainnet protocol state | Fuzzing complex local deployments        |

If you need to fuzz a deployment that exists only in your local environment (for example, a
multi-step Foundry broadcast against Anvil), genesis state is the lighter-weight choice. If the
system you are testing composes with contracts that are deployed on a real network, fork mode is the
right tool.

## Workflow

### 1. Pin a block number

Fork mode queries the network for the state at exactly one block, so you must provide a concrete
block number — block tags such as `LATEST` are not supported. Choose a recent block and record it:
reusing the same block across runs makes the RPC cache effective and keeps your fuzzing campaigns
reproducible.

The block you pin must be served with **state** data, not just headers and receipts. Most public RPC
endpoints only serve state for the most recent ~128 blocks; querying an older block against such an
endpoint will fail. For older blocks, use an archive node or an endpoint with archive access.

### 2. Enable fork mode in `medusa.json`

Add a `forkConfig` object to the `chainConfig` section of your `medusa.json`:

```json
{
  "chainConfig": {
    "forkConfig": {
      "forkModeEnabled": true,
      "rpcUrl": "https://eth-mainnet.g.alchemy.com/v2/<your-api-key>",
      "rpcBlock": 22741099,
      "poolSize": 20
    }
  }
}
```

- **`forkModeEnabled`** — set to `true` to turn fork mode on.
- **`rpcUrl`** — the JSON-RPC endpoint that Medusa queries for remote state. Prefer a private
  endpoint with an API key over a free public gateway; campaigns can issue many queries.
- **`rpcBlock`** — the block number whose state the fork is based on (see step 1).
- **`poolSize`** — the number of RPC clients in the connection pool. A pool size of 2-3x your
  [`workers`](../project_configuration/fuzzing_config.md) count is a good starting point; reduce it
  if you hit rate limits.

### 3. Deploy your contracts on top of the fork

Fork mode changes the state seed, not the deployment flow. Medusa still deploys the contracts from
your compilation artifacts exactly as it does normally — configure
[`targetContracts`](../project_configuration/fuzzing_config.md#targetcontracts), constructor args, and balances as usual. The only
difference is that when your contracts (or the fuzzer) touch state belonging to the forked network,
that state is resolved through the RPC.

This means your deployed contracts can interact with real on-chain protocols. For example, a vault
deployed by Medusa can deposit into the real WETH contract, swap against the real Uniswap pools, and
read the real Chainlink feeds — at their well-known mainnet addresses, with no mocking required.

### 4. Interact with real contracts from your tests

Pre-existing network contracts are not deployed by Medusa, so the fuzzer has no compilation artifact
for them and will not generate calls to them directly. Instead, reference them from your test
contracts by address using a Solidity interface. State for those addresses is fetched from the RPC
the moment execution reads or writes it.

```solidity
interface IERC20 {
    function balanceOf(address account) external view returns (uint256);
}

contract VaultInvariants {
    // Real mainnet contracts, referenced by their well-known addresses.
    IERC20 constant USDC = IERC20(0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48);

    Vault internal vault; // your contract, deployed by Medusa on top of the fork

    constructor() {
        vault = new Vault(address(USDC));
    }

    // The vault's accounting must never exceed the real USDC balance it holds.
    function invariant_vaultAccounting() public {
        assert(vault.totalDeposits() <= USDC.balanceOf(address(vault)));
    }
}
```

For contracts that should be deployed by _you_ at an address of your choosing (rather than read
from the network), see [`predeployedContracts`](../project_configuration/fuzzing_config.md#predeployedcontracts) — it composes naturally
with fork mode.

### 5. Run Medusa

```bash
medusa fuzz
```

During startup, Medusa dials the RPC endpoint once per client in the pool, so an invalid or
unreachable `rpcUrl` fails the campaign immediately rather than mid-run. From here on, fuzzing works
exactly as it does locally: workers clone the base chain, generate call sequences, and check your
properties. Whenever a worker reaches state that lives on the remote network, it is fetched through
the client pool and shared with the other workers through the cache.

## The RPC Cache

Every piece of remote state Medusa fetches is written to a persistent, per-project cache stored in a
`.medusacache` directory next to your `medusa.json`. The cache is keyed by RPC endpoint and block
number, so a second campaign against the same fork — even weeks later — skips the queries it has
already seen. This is the main reason to pin a stable block number rather than chasing the chain
tip: a fixed fork turns an expensive first run into cheap reruns.

The cache is a performance feature only and can be deleted at any time to reclaim disk space. Fork
mode remains correct without it; deleting it simply means the next campaign re-queries the remote
endpoint for state it no longer has.

## Limitations and Practical Tips

- **`rpcBlock` must be a concrete block number.** Block tags like `LATEST` are not supported yet.
  Automate your setup to resolve the latest block once (for example with `cast block-number`) and
  write the result into your configuration if you want a recent fork.
- **Archive data is not free.** Public endpoints generally serve state only for recent blocks. If
  your pinned block is older than the provider's state window, use an archive endpoint.
- **Watch your rate limits.** Parallel workers issue parallel queries. Scale `poolSize` with your
  worker count (2-3x is a sane default), and back off if the endpoint starts rejecting requests.
- **The simulated chain always reports chain ID `1`.** If you fork a non-mainnet network, keep in
  mind that `block.chainid` inside your contracts will still be `1`, which can matter for code that
  includes the chain ID in signatures or domain separators.
- **Time manipulation affects the local chain only.** Cheatcodes such as
  [`vm.warp`](../cheatcodes/warp.md) and [`vm.roll`](../cheatcodes/roll.md) advance Medusa's
  simulated chain; they do not change which block the fork was based on.
- **Top-level calls come from Medusa's own accounts.** The fork imports state, but the senders
  Medusa fuzzes with are its own funded accounts — the fuzzer never signs transactions as a real
  on-chain address. To exercise a privileged entry point of a network contract (for example, calling
  a governance function from its timelock), use [`vm.prank`](../cheatcodes/prank.md) inside one of
  your test contracts to spoof the privileged caller for that call.
