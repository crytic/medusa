# Debugging and Development

## Debugging

The following scripts are available for Medusa developers for debugging changes to the fuzzer.

### Corpus diff

The corpus diff script is used to compare two corpora and identify the methods that are present in one but not the other. This is useful for identifying methods that are missing from a corpus that should be present.

```shell
python3 scripts/corpus_diff.py corpus1 corpus2
```

```shell
Methods only in ~/corpus1:
-  clampSplitWeight(uint32,uint32)

Methods only in ~/corpus2:
  <None>
```

### Corpus stats

The corpus stats script is used to generate statistics about a corpus. This includes the number of sequences, the average length of sequences, and the frequency of methods called.

```shell
python3 scripts/corpus_stats.py corpus
```

```shell
Number of Sequences in ~/corpus: 130

Average Length of Transactions List: 43

Frequency of Methods Called:
-  testReceiversReceivedSplit(uint8): 280
-  setMaxEndHints(uint32,uint32): 174
-  setStreamBalanceWithdrawAll(uint8): 139
-  giveClampedAmount(uint8,uint8,uint128): 136
-  receiveStreamsSplitAndCollectToSelf(uint8): 133
-  testSqueezeViewVsActual(uint8,uint8): 128
-  testSqueeze(uint8,uint8): 128
-  testSetStreamBalance(uint8,int128): 128
-  addStreamWithClamping(uint8,uint8,uint160,uint32,uint32,int128): 125
-  removeAllSplits(uint8): 118
-  testSplittableAfterSplit(uint8): 113
-  testSqueezableVsReceived(uint8): 111
-  testBalanceAtInFuture(uint8,uint8,uint160): 108
-  testRemoveStreamShouldNotRevert(uint8,uint256): 103
-  invariantWithdrawAllTokensShouldNotRevert(): 103
-  collect(uint8,uint8): 101
-  invariantAmtPerSecVsMinAmtPerSec(uint8,uint256): 98
-  testSqueezableAmountCantBeWithdrawn(uint8,uint8): 97
-  split(uint8): 97
-  invariantWithdrawAllTokens(): 95
-  testReceiveStreams(uint8,uint32): 93
-  invariantAccountingVsTokenBalance(): 92
-  testSqueezeWithFuzzedHistoryShouldNotRevert(uint8,uint8,uint256,bytes32): 91
-  testSqueezableAmountCantBeUndone(uint8,uint8,uint160,uint32,uint32,int128): 87
-  testCollect(uint8,uint8): 86
-  testSetStreamBalanceWithdrawAllShouldNotRevert(uint8): 86
-  testAddStreamShouldNotRevert(uint8,uint8,uint160,uint32,uint32,int128): 85
-  testReceiveStreamsShouldNotRevert(uint8): 84
-  addSplitsReceiver(uint8,uint8,uint32): 84
-  setStreamBalanceWithClamping(uint8,int128): 82
-  addSplitsReceiverWithClamping(uint8,uint8,uint32): 80
-  testSetStreamBalanceShouldNotRevert(uint8,int128): 80
-  testSplitShouldNotRevert(uint8): 80
-  squeezeAllAndReceiveAndSplitAndCollectToSelf(uint8): 79
-  addStreamImmediatelySqueezable(uint8,uint8,uint160): 79
-  testSetSplitsShouldNotRevert(uint8,uint8,uint32): 78
-  invariantSumAmtDeltaIsZero(uint8): 78
-  testReceiveStreamsViewConsistency(uint8,uint32): 76
-  squeezeToSelf(uint8): 74
-  collectToSelf(uint8): 72
-  setStreams(uint8,uint8,uint160,uint32,uint32,int128): 70
-  receiveStreamsAllCycles(uint8): 69
-  invariantWithdrawShouldAlwaysFail(uint256): 68
-  addStream(uint8,uint8,uint160,uint32,uint32,int128): 68
-  squeezeWithFuzzedHistory(uint8,uint8,uint256,bytes32): 67
-  setStreamsWithClamping(uint8,uint8,uint160,uint32,uint32,int128): 67
-  splitAndCollectToSelf(uint8): 67
-  testSqueezeWithFullyHashedHistory(uint8,uint8): 65
-  give(uint8,uint8,uint128): 65
-  setSplits(uint8,uint8,uint32): 65
-  testSqueezeTwice(uint8,uint8,uint256,bytes32): 65
-  testSetStreamsShouldNotRevert(uint8,uint8,uint160,uint32,uint32,int128): 64
-  squeezeAllSenders(uint8): 63
-  removeStream(uint8,uint256): 62
-  testCollectableAfterSplit(uint8): 58
-  testCollectShouldNotRevert(uint8,uint8): 56
-  testReceiveStreamsViewVsActual(uint8,uint32): 55
-  receiveStreams(uint8,uint32): 55
-  setSplitsWithClamping(uint8,uint8,uint32): 55
-  testGiveShouldNotRevert(uint8,uint8,uint128): 47
-  setStreamBalance(uint8,int128): 47
-  squeezeWithDefaultHistory(uint8,uint8): 45
-  testSplitViewVsActual(uint8): 45
-  testAddSplitsShouldNotRevert(uint8,uint8,uint32): 30
-  testSqueezeWithDefaultHistoryShouldNotRevert(uint8,uint8): 23

Number of Unique Methods: 65
```
