// This contract verifies the fuzzer captures return values of functions
contract TestContract {
   function testReturnValues(address value) public returns(uint64,uint256,int64,bool,address,bytes32,string memory) {
        bytes32 b = 0x68656c6c6f000000000000000000000000000000000000000000000000000000;
        string memory str = "Hello World!";
        return (1,2,-1,true,address(0),b,str);
    }
}
