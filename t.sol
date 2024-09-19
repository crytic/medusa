// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;
interface ITransientStorageTest {
    function x() external view returns (uint);
    
}
contract B {
    function Z() external {
        if (ITransientStorageTest(msg.sender).x() == 0) {
            revert();
        } else {
            return;
        }
        
    }
 }
contract TransientStorageTest {
    uint public x;
    function useTransientStorage(uint256 data) public {
        B b = new B();
        try b.Z() {

        } catch  {
            x++;
        }
        if (x > 2) {
            x++;
        }
    }

}
