contract TestContract {
    uint256[] public myArray;
    function popEmptyArray() public {
        myArray.pop(); // Pop from empty array
    }
}