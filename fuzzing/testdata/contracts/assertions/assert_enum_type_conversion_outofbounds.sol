// Enum type conversion out of bounds would cause a panic
// PanicCodeEnumTypeConversionOutOfBounds = 0x21
contract TestContract {
    enum MyEnum { A, B, C }

    function enumTypeConversionOutOfBounds() public {
        uint8 value = 4; // Out of bounds for MyEnum
        MyEnum myEnum = MyEnum(value);
    }
}
