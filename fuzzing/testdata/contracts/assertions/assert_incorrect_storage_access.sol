// This contract triggers an incorrect storage access panic
// PanicCodeIncorrectStorageAccess = 0x22

contract TestContract {
    uint256[] public myArray;

    function incorrectStorageAccess() public returns(uint256) {
         uint256 index = 7;  // Index out of bounds
         return myArray[index];  // Incorrect storage access
    }
}
