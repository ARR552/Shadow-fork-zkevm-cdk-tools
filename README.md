# Shadow fork tools

Usefull commands:

```shell
npx hardhat help
npx hardhat test
REPORT_GAS=true npx hardhat test
npx hardhat node
npx hardhat ignition deploy ./ignition/modules/Lock.js
```
npx hardhat node --fork https://sepolia.infura.io/v3/<key> --fork-block-number 6849976



```
anvil --block-time 12 --port 8545 --fork-url https://sepolia.infura.io/v3/<your_key> --fork-block-number 6787526

ganache --chain.chainId 11155111 --chain.hardfork shanghai --miner.blockTime 12 --wallet.mnemonic "test test test test test test test test test test test junk" --fork.url https://sepolia.infura.io/v3/<key> --fork.blockNumber 6849976 --server.host 0.0.0.0 --server.port 8545 --wallet.unlockedAccounts 0xff6250d0E86A2465B0C1bF8e36409503d6a26963 0x761d53b47334bEe6612c0Bd1467FB881435375B2

forge test --rpc-url http://127.0.0.1:8545/ --evm-version shanghai --match-path test/SendSequence.test.sol -vvv

forge script --rpc-url http://127.0.0.1:8545/ --evm-version shanghai -vvv test/SendSequence.script.sol
```

The L1 network can be shadow forked using anvil or hardhat command.

The file `./sendImpersonateTransfer.js` shows a way to impersonate an account in a script. L1 state is not modified. Once the script ends the state is reverted.


CDK_ENVIRONMENT=cardona docker compose -f docker-compose.yml up -d zkevm-state-db
CDK_ENVIRONMENT=cardona docker compose -f docker-compose.yml up -d zkevm-pool-db
CDK_ENVIRONMENT=cardona docker compose -f docker-compose.yml up -d zkevm-prover
CDK_ENVIRONMENT=cardona docker compose -f docker-compose.yml up -d cdk-erigon
CDK_ENVIRONMENT=cardona docker compose -f docker-compose.yml up -d zkevm-seqsender
CDK_ENVIRONMENT=cardona docker compose -f docker-compose.yml up -d zkevm-pool-manager
CDK_ENVIRONMENT=cardona docker compose -f docker-compose.yml up -d zkevm-shadow-fork
CDK_ENVIRONMENT=cardona docker compose -f docker-compose.yml up -d zkevm-ssender
CDK_ENVIRONMENT=cardona docker compose -f docker-compose.yml up -d zkevm-aggregator

docker compose -f docker-compose.yml down --remove-orphans

docker compose -f docker-compose.yml stop zkevm-state-db && docker compose -f docker-compose.yml rm -f zkevm-state-db
docker compose -f docker-compose.yml stop zkevm-pool-db && docker compose -f docker-compose.yml rm -f zkevm-pool-db
docker compose -f docker-compose.yml stop zkevm-prover && docker compose -f docker-compose.yml rm -f zkevm-prover
docker compose -f docker-compose.yml stop cdk-erigon && docker compose -f docker-compose.yml rm -f cdk-erigon
docker compose -f docker-compose.yml stop zkevm-seqsender && docker compose -f docker-compose.yml rm -f zkevm-seqsender
docker compose -f docker-compose.yml stop zkevm-pool-manager && docker compose -f docker-compose.yml rm -f zkevm-pool-manager
docker compose -f docker-compose.yml stop zkevm-shadow-fork && docker compose -f docker-compose.yml rm -f zkevm-shadow-fork
docker compose -f docker-compose.yml stop zkevm-ssender && docker compose -f docker-compose.yml rm -f zkevm-ssender
docker compose -f docker-compose.yml stop zkevm-aggregator && docker compose -f docker-compose.yml rm -f zkevm-aggregator
