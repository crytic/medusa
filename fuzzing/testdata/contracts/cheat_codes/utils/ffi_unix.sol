interface CheatCodes {
    function ffi(string[] calldata) external returns (bytes memory);
}

contract TestContract {
    CheatCodes cheats;

    constructor() {
        cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    }

    function testABIDecode() public {
        // Create command
        string[] memory inputs = new string[](3);
        inputs[0] = "echo";
        inputs[1] = "-n";
        // Encoded "gm"
        inputs[2] = "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000568656C6C6F000000000000000000000000000000000000000000000000000000";

        // Call cheats.ffi
        bytes memory res = cheats.ffi(inputs);

        // ABI decode
        string memory output = abi.decode(res, (string));
        assert(keccak256(abi.encodePacked(output)) == keccak256(abi.encodePacked("hello")));
    }

    function testUTF8() public {
        // Create command
        string[] memory inputs = new string[](3);
        inputs[0] = "echo";
        inputs[1] = "-n";
        inputs[2] = "hello";

        // Call cheats.ffi
        bytes memory res = cheats.ffi(inputs);

        // Convert to UTF-8 string
        string memory output = string(res);
        assert(keccak256(abi.encodePacked(output)) == keccak256(abi.encodePacked("hello")));
    }
}
