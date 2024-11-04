require("@nomicfoundation/hardhat-toolbox");

module.exports = {
  solidity: "0.8.17",
  networks: {
    hardhat: {
      forking: {
        url: "https://sepolia.infura.io/v3/<your_key>",
        blockNumber: 6849976
      },
      mining: {
        auto: false,
        interval: 12000
      },
      chainId: 11155111,
    },
  },
};