# Cheatcodes Overview

Cheatcodes allow users to manipulate EVM state, blockchain behavior, provide easy ways to manipulate data, and much more.
The cheatcode contract is deployed at `0x7109709ECfa91a80626fF3989D68f67F5b1DD12D`.

## Cheatcode Interface

The following interface must be added to your Solidity project if you wish to use cheatcodes. Note that if you use Foundry
as your compilation platform that the cheatcode interface is already provided [here](https://book.getfoundry.sh/reference/forge-std/#forge-stds-test).
However, it is important to note that medusa does not support all the cheatcodes provided out-of-box
by Foundry (see below for supported cheatcodes).

```solidity
interface StdCheats {
    // Set block.timestamp
    function warp(uint256) external;

    // Set block.number
    function roll(uint256) external;

    // Set block.basefee
    function fee(uint256) external;

    // Set block.difficulty and block.prevrandao
    function difficulty(uint256) external;

    // Set block.chainid
    function chainId(uint256) external;

    // Sets the block.coinbase
    function coinbase(address) external;

    // Loads a storage slot from an address
    function load(address account, bytes32 slot) external returns (bytes32);

    // Stores a value to an address' storage slot
    function store(address account, bytes32 slot, bytes32 value) external;

    // Sets the *next* call's msg.sender to be the input address
    function prank(address) external;

    // Set msg.sender to the input address until the current call exits
    function prankHere(address) external;

    // Sets an address' balance
    function deal(address who, uint256 newBalance) external;

    // Sets an address' code
    function etch(address who, bytes calldata code) external;

    // Signs data
    function sign(uint256 privateKey, bytes32 digest)
        external
        returns (uint8 v, bytes32 r, bytes32 s);

    // Computes address for a given private key
    function addr(uint256 privateKey) external returns (address);

    // Gets the nonce of an account
    function getNonce(address account) external returns (uint64);

    // Sets the nonce of an account
    // The new nonce must be higher than the current nonce of the account
    function setNonce(address account, uint64 nonce) external;

    // Performs a foreign function call via terminal
    function ffi(string[] calldata) external returns (bytes memory);

    // Take a snapshot of the current state of the EVM
    function snapshot() external returns (uint256);

    // Revert state back to a snapshot
    function revertTo(uint256) external returns (bool);

    // Convert Solidity types to strings
    function toString(address) external returns(string memory);
    function toString(bytes calldata) external returns(string memory);
    function toString(bytes32) external returns(string memory);
    function toString(bool) external returns(string memory);
    function toString(uint256) external returns(string memory);
    function toString(int256) external returns(string memory);

    // Convert strings into Solidity types
    function parseBytes(string memory) external returns(bytes memory);
    function parseBytes32(string memory) external returns(bytes32);
    function parseAddress(string memory) external returns(address);
    function parseUint(string memory)external returns(uint256);
    function parseInt(string memory) external returns(int256);
    function parseBool(string memory) external returns(bool);
}
```

# Using cheatcodes

Below is an example snippet of how you would import the cheatcode interface into your project and use it.

```solidity
// Assuming cheatcode interface is in the same directory
import "./IStdCheats.sol";

// MyContract will utilize the cheatcode interface
contract MyContract {
    // Set up reference to cheatcode contract
    IStdCheats cheats = IStdCheats(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    // This is a test function that will set the msg.sender's nonce to the provided input argument
    function testFunc(uint256 _x) public {
        // Ensure that the input argument is greater than msg.sender's current nonce
        require(_x > cheats.getNonce(msg.sender));

        // Set sender's nonce
        cheats.setNonce(msg.sender, x);

        // Assert that the nonce has been correctly updated
        assert(cheats.getNonce(msg.sender) == x);
    }
}
```
