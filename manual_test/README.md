# Manual Verification: Native anvil_dumpState Support

This directory contains a minimal test case to verify that Medusa can load native `anvil_dumpState` output without preprocessing.

## What This Tests

- Native `anvil_dumpState` format support (gzip-compressed hex)
- State preservation across anvil → medusa workflow
- Contract deployment state is correctly loaded
- ABI mapping for genesis-loaded contracts (genesisContractMappings)

## Prerequisites

```bash
# Required tools
forge --version  # Foundry
cast --version   # Foundry
anvil --version  # Foundry
go version       # Go (to build medusa)
```

No additional dependencies required - this is a standalone test!

## Manual Verification Steps

### 1. Start Fresh Anvil Instance

```bash
cd manual_test

# Kill any existing anvil instances
pkill anvil 2>/dev/null

# Start anvil on default port
anvil --port 8545 &
sleep 2
```

### 2. Deploy Contract to Anvil

```bash
# Deploy using Foundry script
forge script script/Deploy.s.sol:DeployScript \
  --rpc-url http://127.0.0.1:8545 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
  --broadcast

# You should see:
# - "Counter deployed at: 0x5FbDB2315678afecb367f032d93F642f64180aa3"
# - "Current count: 45" (42 initial + 3 increments)
```

### 3. Capture State with anvil_dumpState (NO PREPROCESSING!)

```bash
# Capture raw anvil state - this is the key feature we're testing
cast rpc anvil_dumpState --rpc-url http://127.0.0.1:8545 > anvil_dump.json

# Verify it's the compressed format (starts with "0x1f8b08")
head -c 50 anvil_dump.json
# Should see: "0x1f8b08000000000000ff...
```

### 4. Load State into Medusa

```bash
# Build medusa (from repo root)
cd ..
go build -o medusa_fix469

# Run medusa with the native anvil dump
cd manual_test
../medusa_fix469 fuzz --config medusa.json

# Expected output:
# ✓ "Loaded genesis state from anvil_dump.json with N accounts"
# ✓ "Mapped genesis contract at 0x5FbDB...0aa3 to Counter"
# ✓ Fuzzer runs successfully
# ✓ Property test "echidna_count_reasonable" is tested
# ✓ Functions like "increment()" and "decrement()" are fuzzed
```

### 5. Verify State Was Loaded Correctly

The fuzzer should:
- Load the Counter contract at the deployed address (0x5FbDB2315678afecb367f032d93F642f64180aa3)
- Map the address to the Counter ABI using genesisContractMappings
- Preserve the count value (45) from deployment
- Successfully fuzz the contract methods (increment, decrement)
- Test the property function (echidna_count_reasonable)

**Key Feature**: The `genesisContractMappings` configuration tells Medusa which compiled
contract corresponds to each genesis address, allowing it to know which functions to call.

### 6. Cleanup

```bash
# Stop anvil
pkill anvil

# Remove generated files
rm -rf anvil_dump.json out cache broadcast
```

## Success Criteria

✅ **State Loading:** Raw `cast rpc anvil_dumpState > file.json` output works directly
✅ **ABI Mapping:** Genesis contracts are mapped to ABIs via genesisContractMappings
✅ **Fuzzing Works:** Medusa can call functions on genesis-loaded contracts
✅ **Property Testing:** Property tests run on genesis-loaded contracts
✅ **State Preservation:** Contract state from deployment is correctly loaded

## File Format Comparison

**Native anvil_dumpState (now supported):**
```json
"0x1f8b08000000000000ff..."  // Gzip-compressed hex string
```

**Old format (still supported):**
```json
{
  "0xADDRESS": {
    "balance": "0x0",
    "nonce": "0x5",
    "code": "0x...",
    "storage": {}
  }
}
```

Both formats work automatically - Medusa detects which one you're using!
