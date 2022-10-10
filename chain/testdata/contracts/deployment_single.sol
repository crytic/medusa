// This contract is used to test replayability of messages/blocks to ensure that state changing operations work the
// same.
contract StateChangingTest {
    uint x;
    uint y;

    function setX() public {
        x++;
    }

    function setY() public {
        y++;
    }
}
