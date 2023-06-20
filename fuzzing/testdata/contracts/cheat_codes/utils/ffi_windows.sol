interface CheatCodes {
    function ffi(string[] calldata) external returns (bytes memory);
}

contract TestContract {
    CheatCodes cheats;

    constructor() {
        cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    }

    function testABIDecode() public {
        // Create command (on windows is a shell command that prints a new line, so this is an ugly hack to remove it)
        string[] memory inputs = new string[](3);
        inputs[0] = "cmd";
        inputs[1] = "/C";
        // Encoded "gm"
        inputs[2] = "echo|set /p dummyVar=0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000568656C6C6F000000000000000000000000000000000000000000000000000000";

        // Call cheats.ffi
        bytes memory res = cheats.ffi(inputs);

        // ABI decode
        string memory output = abi.decode(res, (string));
        assert(keccak256(abi.encodePacked(output)) == keccak256(abi.encodePacked("hello")));
    }

    function testUTF8() public {
        // Create command (on windows is a shell command that prints a new line, so this is an ugly hack to remove it)
        string[] memory inputs = new string[](3);
        inputs[0] = "cmd";
        inputs[1] = "/C";
        inputs[2] = "echo|set /p dummyVar=hello";

        // Call cheats.ffi
        bytes memory res = cheats.ffi(inputs);

        // Convert to UTF-8 string
        string memory output = string(res);
        assert(keccak256(abi.encodePacked(output)) == keccak256(abi.encodePacked("hello")));
    }
}
