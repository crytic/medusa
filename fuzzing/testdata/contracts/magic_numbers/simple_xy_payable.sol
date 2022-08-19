contract TestXYPayable {
    uint paidAmount;
    uint paidAmount2;
    bool calledOnce;

    function p1() public payable {
        paidAmount = msg.value;
    }

    function p2() public payable {
        paidAmount2 = msg.value;
    }

    function calledAtLeastOnce() public {
        calledOnce = true;
    }

    function medusa_never_pay_exact_amounts() public view returns (bool) {
        // ASSERTION: p1 will never be last paid 7777, p2 8888, when calledAtLeastOnce was called.
        return !(paidAmount == 7777 && paidAmount2 == 8888 && calledOnce);
    }
}
