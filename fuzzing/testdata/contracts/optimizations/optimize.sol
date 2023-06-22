contract TestContract {
  int256 input;

  function expensive(uint8 n) public {
    for (uint8 i = 0; i < n; i++)
      keccak256(abi.encode(n));
  }

  function set(int256 _input) public {
    input = _input;
  }

  function optimize_opt_linear() public view returns (int256) {
    if (input > -4242)
      return -input;
    else
      return 0;
  }
}
