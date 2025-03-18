// This test ensures that startPrank pranks delegate calls accordingly, so the code address and msg.sender is both set.
interface CheatCodes {
    function startPrank(address msgSender) external;
    function startPrank(address msgSender, bool delegateCall) external;

    function stopPrank() external;
}

contract TestBaseContract {
    address public addr;

    function getAddr() public {
        addr = msg.sender;
    }
}

contract TestProxyContract {
    address public addr;

    function getAddr(address _logic) public returns(address,address) {
        (bool success, ) = _logic.delegatecall(abi.encodeWithSignature("getAddr()"));
        assert(success);
        return (msg.sender, addr);
    }

    function getResultAddr() public returns (address) {
        return addr;
    }
}

contract TestContract {
    address public addr;
    TestContract thisExternal = TestContract(address(this));
    CheatCodes vm = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    TestBaseContract baseContract = new TestBaseContract();
    TestProxyContract proxyContract = new TestProxyContract();

    function test_startPrank() public {

        // Without pranking delegate, msg.sender will be set to proxyContract, but the CodeAddress will remain as
        // this, thus THIS contract's storage is modified.
        vm.startPrank(address(proxyContract), false);
        (bool success, ) = address(baseContract).delegatecall(abi.encodeWithSignature("getAddr()")); // sets `addr` in proxy
        assert(success);
        assert(addr == address(proxyContract)); // `addr` should be set in proxy now.
        vm.stopPrank();

        // When pranking delegate, we should expect msg.sender and the CodeAddress to be set, so storage in the
        // ProxyContract should be upgraded.
        vm.startPrank(address(proxyContract), true);
        (success, ) = address(baseContract).delegatecall(abi.encodeWithSignature("getAddr()")); // sets `addr` in proxy
        assert(success);
        assert(proxyContract.getResultAddr() == address(proxyContract)); // `addr` should be set in proxy now.
        vm.stopPrank();
    }
}