# Corpus
We have an in-memory `Corpus` struct shared between corpus workers, which syncs to disk. `corpus_files.go` handles loading/writing. Each call sequence is represented as a json file in a corpus directory. There is a call_sequences directory for call sequences that do not trigger tests, and a test_results directory for call sequences that do. Files are not written by default until `corpusDirectory.writeFiles()` is called. There is a legacy "immutable/mutable" file format that `Corpus.migrateLegacyCorpus()` handles. When working with `Corpus`, make sure to use `callSequencesLock` when necessary.
## Corpus pruner
The `CorpusPruner` struct is used to handle corpus pruning jobs which happen once every `pruneFrequency` minutes. This job removes "redundant" corpus items that no longer add any coverage. During pruning, we randomize the order of the corpus, then go through it one-by-one, running call sequences and seeing if any new coverage resulted. The randomization is to handle cases where, for example, call sequence #1 adds coverage A and call sequence #2 adds coverage A *and* B. In this case call sequence #1 is redundant and should be removed, but would not be removed by our algo unless randomization switches the order.
This results in a smaller, more minimal/effective corpus, lower memory usage, and possibly better performance.
The corpus pruner doesn't remove call sequences from disk, only from memory.
See `corpus_pruner.go` and `Corpus.PruneSequences`.
## Corpus cleaner
The corpus cleaner removes old, invalid call sequences from the corpus directory on disk. It is only triggered by the `medusa corpus clean` command and is not run during normal fuzzing.
See `corpus_cleaner.go` and `Corpus.CleanInvalidSequences`.
