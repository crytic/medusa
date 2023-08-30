// This contract verifies two different, but specific payable calls will be provided by the fuzzer.
contract TestContract {
    uint paidAmount;
    uint paidAmount2;

    function p1() public payable {
        paidAmount = msg.value;
    }

    function p2() public payable {
        paidAmount2 = msg.value;
    }

    function property_never_pay_exact_amounts() public view returns (bool) {
        // ASSERTION: paid amounts should never equal the exact numbers below.
        return !(paidAmount == 7777 && paidAmount2 == 8888);
    }
}
