// This contract ensures that we can target or exclude functions
contract TestContract {
    uint odd_counter = 1;
    uint even_counter = 2;
    event Counter(uint256 value);
    function f() public {
        odd_counter += 1;
        emit Counter(odd_counter);
    }

    function g() public {
        even_counter += 2;
        emit Counter(even_counter);

    }

    function h() public {
        odd_counter += 3;
        emit Counter(odd_counter);

    }

    function i() public {
        even_counter += 4;
        emit  Counter(even_counter);
    }

    function property_a() public view returns (bool) {
        return (odd_counter != 100);
    }

    function optimize_b() public view returns (int256) {
        return -1;
    }
}
