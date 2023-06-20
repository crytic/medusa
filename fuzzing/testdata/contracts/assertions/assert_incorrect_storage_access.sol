contract TestContract {
    function incorrectStorageAccess() public {
        mapping(uint256 => uint256) myMapping;
        uint256 value = myMapping[123]; // Incorrect storage access
    }
}