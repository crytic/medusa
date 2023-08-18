// This contract verifies the fuzzer can extract AST literals of different subdenominations from the file.
contract TestContract {
    function addressValues() public {
        address x = 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D;
        assert(x != address(0x1234567890123456789012345678901234567890));
   }
    function uintValues() public {
        // Use all integer denoms
        uint x = 111;
        x = 1 wei;
        x = 2 gwei;
        //x = 3 szabo;
        //x = 4 finney;
        x = 5 ether;
        x = 6 seconds;
        x = 7 minutes;
        x = 8 hours;
        x = 9 days;
        x = 10 weeks;
        //x = 11 years;

        // Dummy assertion that should always pass.
        assert(x != 0);
   }
   function intValues() public {
           // Use all integer denoms
           int x = -111;
           x = -1 wei;
           x = -2 gwei;
           //x = -3 szabo;
           //x = -4 finney;
           x = -5 ether;
           x = -6 seconds;
           x = -7 minutes;
           x = -8 hours;
           x = -9 days;
           x = -10 weeks;
           //x = -11 years;

           // Dummy assertion that should always pass.
           assert(x != 0);
      }
   function stringValues() public {
        string memory s = "testString";
        s = "testString2";
        assert(true);
   }
}
