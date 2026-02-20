# Corpus Prefill Verification

This branch contains verification instructions to help you test the corpus prefilling functionality.

## Verification Steps

### 1. Create a test contract

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Counter {
    uint256 public count;

    function increment() public {
        count++;
    }   

    function decrement() public {
        count--;
    }   

    function set(uint256 _count) public {
        count = _count;
    }   
}

contract CounterTest {
    Counter public counter;

    function setUp() public {
        counter = new Counter();
    }   

    function testIncrement() public {
        counter.increment();
        counter.increment();
    }   

    function testDecrement() public {
        counter.set(10);
        counter.decrement();
    }   

    function testMultiple() public {
        counter.increment();
        counter.set(5);
        counter.decrement();
    }   
}
```

### 2. Create a config with prefillCorpus enabled

```json
{
  "fuzzing": {
    "prefillCorpus": true,
    "corpusDirectory": "./corpus_test",
    "timeout": 10,
    "targetContracts": ["Counter"]
  },
  "compilation": {
    "target": "test.sol"
  }
}
```

**Important:** `targetContracts` must specify the contract(s) to fuzz (e.g., `Counter`), NOT the test contract (`CounterTest`).

### 3. Run medusa

```bash
go build
./medusa fuzz --config config.json
```

### 4. Check the logs

Look for these messages:
```
[corpus-prefill] Attempting to prefill corpus from Foundry tests
[corpus-prefill] Successfully prefilled corpus with N test sequences
```

**Should NOT see:**
```
Python 3 or Slither not found  # This was the old implementation
```

### 5. Verify corpus files

```bash
ls corpus_test/
cat corpus_test/call_sequences_0.json | jq .
```

You should see JSON files with extracted call sequences.

## Expected Results

✅ **Success indicators:**
- Log message: "Successfully prefilled corpus with N test sequences" (N = number of test functions)
- Corpus directory created at `./corpus_test/`
- JSON files in corpus directory containing call sequences
- Fuzzer starts and runs normally

## Implementation Details

The new implementation:
1. Uses the AST that Medusa already obtains from `solc --combined-json ast`
2. Parses the AST in pure Go to find test contracts and functions
3. Extracts external function calls from test function bodies
4. Converts them to CallSequence format for the corpus
5. Uses case-insensitive matching so variable names (e.g., "counter") match contract names (e.g., "Counter")

See the commit message for detailed technical information.

## Troubleshooting

### "No test sequences found"

Check that:
- Your contract name contains "Test" (e.g., `CounterTest`)
- Your functions start with "test", "invariant_", or "testfuzz"
- Your test functions make external contract calls (e.g., `counter.increment()`)
- Variable names are close to contract names (case doesn't matter)

### Build fails

```bash
go build
# Should compile without errors
```

### No corpus files created

Check that:
- `prefillCorpus: true` in your config
- `corpusDirectory` is set in your config
- `targetContracts` is set in your config
- Compilation was successful (contracts compiled)

## Questions?

If you have questions or find issues, please comment on the PR or open an issue.
