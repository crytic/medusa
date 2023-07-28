library Library1 {
    function getLibrary() public returns(string memory) {
        return "Library";
    }
}
library Library2 {
    function getLibrary() public returns(string memory) {
        Library1.getLibrary();
    }
}
library Library3 {
    function getLibrary() public returns(string memory) {
        Library2.getLibrary();
        Library1.getLibrary();
    }
}
contract TestExternalLibrary {
    function test() public {
        Library3.getLibrary();

    }
    function fuzz_me() public returns(bool){
        if (keccak256(abi.encodePacked(Library3.getLibrary())) == keccak256(abi.encodePacked("Library"))) {
            return false;            
        }
    }
}