contract FailedCompilationContract {
    uint512 x; // this type doesn't actually exist and should cause a compilation error.

    function setX(uint value) public {
        x = value;
    }
}
