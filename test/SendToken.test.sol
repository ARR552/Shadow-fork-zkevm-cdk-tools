// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import {Test, console} from "./forge-std/Test.sol";

// ERC-20 interface for interacting with the pol token
interface IERC20 {
    function balanceOf(address) external view returns (uint256);
    function transfer(address, uint256) external returns (bool);
    // function transferFrom(address, address, uint256) external returns (bool);
    function decimals() external view returns (uint8);
}

// TokenTransferTest is a contract that sets up and runs the test
contract TokenTransferTest is Test {
    IERC20 pol; // Interface instance for pol
    address whale = 0x761d53b47334bEe6612c0Bd1467FB881435375B2; // Sequencer account
    address recipient = 0xBc3cD9B0933340faAcdF4E743197E0ceA7FC6Dfb; // Random account
    address polAddress = 0x1850Dd35dE878238fb1dBa7aF7f929303AB6e8E4;

    // setUp function runs before each test, setting up the environment
    function setUp() public {
        pol = IERC20(polAddress); // Initialize the Pol contract interface

        // Impersonate the whale account for testing
        vm.startPrank(whale);
    }

    // testTokenTransfer function tests the transfer of pol from the whale account to the recipient
    function testTokenTransfer() public {
        uint256 initialBalanceSender = pol.balanceOf(whale);
        console.log("initialBalanceSender: ", initialBalanceSender);
        uint256 initialBalance = pol.balanceOf(recipient); // Get the initial balance of the recipient
        uint8 decimals = pol.decimals(); // Get the decimal number of pol
        uint256 transferAmount = 1 ** decimals;
        console.log("Recipient's initial balance: ", initialBalance); // Log the initial balance to the console

        // Perform the token transfer from the whale to the recipient
        pol.transfer(recipient, transferAmount);
        // pol.transferFrom(whale, recipient, transferAmount);

        uint256 finalBalance = pol.balanceOf(recipient); // Get the final balance of the recipient

        console.log("Recipient's final balance: ", finalBalance); // Log the final balance to the console

        // Verify that the recipient's balance increased by the transfer amount
        assertEq(finalBalance, initialBalance + transferAmount, "Token transfer failed");

        vm.stopPrank(); // Stop impersonating the whale account
    }
}