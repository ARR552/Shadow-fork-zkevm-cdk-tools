# Shadow fork tools

Usefull commands:

```shell
npx hardhat help
npx hardhat test
REPORT_GAS=true npx hardhat test
npx hardhat node
npx hardhat ignition deploy ./ignition/modules/Lock.js
```


```
anvil --block-time 12 --port 8545 --fork-url https://sepolia.infura.io/v3/<your_key> --fork-block-number 6787526

forge test --rpc-url http://127.0.0.1:8545/ --evm-version shanghai --match-path test/SendSequence.test.sol -vvv

forge script --rpc-url http://127.0.0.1:8545/ --evm-version shanghai -vvv test/SendSequence.script.sol
```

The L1 network can be shadow forked using anvil or hardhat command.

The file `./sendImpersonateTransfer.js` shows a way to impersonate an account in a script. L1 state is not modified. Once the script ends the state is reverted.


