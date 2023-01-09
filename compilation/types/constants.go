package types

const (
	// LibraryIndicator is used to identify whether a contract is a library. If the runtime bytecode begins with LibraryIndicator
	// then the contract is a library. This indicator is equivalent to a `PUSH20` instruction of a 20-byte zero string.
	LibraryIndicator = "0x730000000000000000000000000000000000000000"

	// LibraryIndicatorLength is the character length of the LibraryIndicator
	LibraryIndicatorLength = len(LibraryIndicator)
)
