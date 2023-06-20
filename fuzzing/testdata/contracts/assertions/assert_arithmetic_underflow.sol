contract TestContract {
    function arithmeticOverflow() public {
        uint8 a = 255;
        uint8 b = 1;
        uint8 c = a + b;
    }
}