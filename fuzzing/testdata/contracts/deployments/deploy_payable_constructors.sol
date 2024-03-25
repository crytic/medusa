// This source file provides two contracts to test whether we are able to send ether to payable constructors. FirstContract
// should get no ether and while SecondContract should receive 1 ether.
contract FirstContract {
    constructor() payable {}

    function property_contract_has_no_balance() public returns(bool) {
        return address(this).balance == 0;
    }
}


contract SecondContract {
    constructor() payable {}

    function property_contract_has_balance() public returns(bool) {
        return address(this).balance == 1 ether;
    }
}
