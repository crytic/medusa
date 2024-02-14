// DeployedContractStartingBalance checks the balance of a contract after deployment to ensure the starting balance was properly set
contract DeployedContractStartingBalance {
    receive() external payable {}

    function checkBalance() public {
        assert(address(this).balance == 3000);
    }
}