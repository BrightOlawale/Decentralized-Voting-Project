module.exports = {
  networks: {
    development: {
      host: "127.0.0.1",
      port: 8545,
      network_id: "*",
      gas: 6721975,
      gasPrice: 20000000000,
    },
    sepolia: {
      network_id: 11155111,
      provider: () =>
        new (require("@truffle/hdwallet-provider"))(
          [process.env.PRIVATE_KEY],
          process.env.SEPOLIA_RPC_URL
        ),
      confirmations: 2,
      timeoutBlocks: 200,
      skipDryRun: true,
    },
  },
  compilers: {
    solc: {
      version: "0.8.19",
      settings: {
        optimizer: {
          enabled: true,
          runs: 200,
        },
      },
    },
  },
  contracts_directory: "./contracts",
  contracts_build_directory: "./build/contracts",
  plugins: ["truffle-plugin-verify"],
  api_keys: { etherscan: process.env.ETHERSCAN_API_KEY },
};
