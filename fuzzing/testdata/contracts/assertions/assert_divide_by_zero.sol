contract TestContract {
    function divideByZero() public {
        uint8 a = 42;
        uint8 b = 0;
        uint8 c = a / b;
    }
}