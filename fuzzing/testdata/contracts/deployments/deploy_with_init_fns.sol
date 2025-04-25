// Ultra-simple test for initialization functions with parameters
contract SimpleInitParamTest {
    // Track if functions were called and parameter values
    bool public initCalled;
    uint public initValue;
    
    // Empty constructor
    constructor() {}
    
    // Initialization function with a parameter
    function initWithParam(uint _value) public {
        initCalled = true;
        initValue = _value;
        emit InitCalled("initWithParam", _value);
    }
    
    // Event for tracking
    event InitCalled(string functionName, uint value);
}