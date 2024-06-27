pragma solidity ^0.8.0;

// This contract includes a function that we will call from TestContract.
contract AnotherContract {
    // This function doesn't need to do anything specific for this example.
    function externalFunction() public pure returns (string memory) {
        return "External function called";
    }
}

// This contract ensures the fuzzer can encounter an immediate assertion failure.
contract TestContract {
    AnotherContract public anotherContract;

    bytes private constant _chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

    function random(uint seed) internal view returns (uint) {
        return uint(keccak256(abi.encodePacked(block.timestamp, block.difficulty, seed)));
    }

    function generateRandomString(uint seed) public view returns (string memory) {
        bytes memory result = new bytes(6);
        uint rand = random(seed);

        for (uint i = 0; i < 6; i++) {
            result[i] = _chars[rand % _chars.length];
            rand = rand / _chars.length;
        }

        return string(result);
    }

    function generateRandomByteArray(uint seed, uint length) public view returns (bytes memory) {
        bytes memory result = new bytes(length);
        uint rand = random(seed);

        for (uint i = 0; i < length; i++) {
            result[i] = _chars[rand % _chars.length];
            rand = rand / _chars.length;
        }

        return result;
    }

    event ValueReceived(uint indexed value, uint second_val, string test, bytes byteArray);
    event ValueNonIndexedReceived(uint firstval, uint secondval, bytes1 myByte);

    function internalFunction() public returns (string memory, string memory, uint, bytes1, bytes memory) {
        string memory internalString = generateRandomString(444);
        uint randInt = 4+14;
        bytes1 randByte = 0x44;
        bytes memory randBytes = generateRandomByteArray(555, 11);
        return ("Internal function called", internalString, randInt, randByte, randBytes);
    }

    // Deploy AnotherContract within the TestContract
    constructor() {
//        internalFunction();
        anotherContract = new AnotherContract();
    }


    function callingMeFails(uint value) public {
        // Call internalFunction()
        internalFunction();
        // Call the external function in AnotherContract.
        anotherContract.externalFunction();

        string memory randString = generateRandomString(123);
        bytes memory byteArray = generateRandomByteArray(456, 10);

        uint second_val = 2+12;

        bytes1 myByte = 0x51;

        emit ValueReceived(value, second_val, randString, byteArray);
        emit ValueNonIndexedReceived(111+111, 444+444, myByte);

        // ASSERTION: We always fail when you call this function.
        assert(false);

    }
}
