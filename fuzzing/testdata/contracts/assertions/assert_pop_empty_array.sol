// Popping from an empty array triggers a `PopEmptyArray` panic
// PanicCodePopEmptyArray = 0x31

contract TestContract {
    uint256[] public myArray;
    function popEmptyArray() public {
        myArray.pop(); // Pop from empty array
    }
}
