// A call to `arithmeticOverflow` function in this contract would trigger an arithmetic overflow panic
// PanicCodeArithmeticUnderOverflow = 0x11
contract TestContract {
    function arithmeticOverflow() public {
        uint8 a = 255;
        uint8 b = 1;
        uint8 c = a + b;
    }
}
