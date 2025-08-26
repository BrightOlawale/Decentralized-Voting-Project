const Web3 = require("web3");

// Configuration
const CONTRACT_ADDRESS = "0xF5B429316e06787683C7f4020533e3D434dad5fb";
const TEST_TERMINAL_ADDRESS = "0xfaaa2d810fEE50582f04b6A741fC0D5389573654";
const OWNER_ADDRESS = "0x627306090abab3a6e1400e9345bc60c78a8bef57"; // First Ganache account

// Connect to local blockchain
const web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:8545"));

// Contract ABI (minimal for authorizeTerminal function)
const contractABI = [
  {
    inputs: [
      {
        internalType: "address",
        name: "_terminal",
        type: "address",
      },
      {
        internalType: "bool",
        name: "_status",
        type: "bool",
      },
    ],
    name: "authorizeTerminal",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "_terminal",
        type: "address",
      },
    ],
    name: "isTerminalAuthorized",
    outputs: [
      {
        internalType: "bool",
        name: "",
        type: "bool",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
];

async function authorizeTerminal() {
  try {
    // Create contract instance
    const contract = new web3.eth.Contract(contractABI, CONTRACT_ADDRESS);

    // Check current authorization status
    console.log("Checking current authorization status...");
    const isAuthorized = await contract.methods
      .isTerminalAuthorized(TEST_TERMINAL_ADDRESS)
      .call();
    console.log(
      `Terminal ${TEST_TERMINAL_ADDRESS} is currently authorized: ${isAuthorized}`
    );

    if (isAuthorized) {
      console.log("Terminal is already authorized!");
      return;
    }

    // Get the owner's private key (first Ganache account)
    const accounts = await web3.eth.getAccounts();
    const ownerAccount = accounts[0]; // First account is the owner

    console.log(`Using owner account: ${ownerAccount}`);

    // Create transaction
    const authorizeData = contract.methods
      .authorizeTerminal(TEST_TERMINAL_ADDRESS, true)
      .encodeABI();

    const tx = {
      from: ownerAccount,
      to: CONTRACT_ADDRESS,
      data: authorizeData,
      gas: 200000,
      gasPrice: web3.utils.toWei("20", "gwei"),
    };

    // Estimate gas
    const gasEstimate = await web3.eth.estimateGas(tx);
    console.log(`Estimated gas: ${gasEstimate}`);
    tx.gas = gasEstimate;

    // Send transaction
    console.log("Authorizing terminal...");
    const result = await web3.eth.sendTransaction(tx);
    console.log(`Transaction hash: ${result.transactionHash}`);

    // Verify authorization
    const newAuthStatus = await contract.methods
      .isTerminalAuthorized(TEST_TERMINAL_ADDRESS)
      .call();
    console.log(
      `Terminal ${TEST_TERMINAL_ADDRESS} is now authorized: ${newAuthStatus}`
    );

    if (newAuthStatus) {
      console.log("✅ Terminal authorization successful!");
    } else {
      console.log("❌ Terminal authorization failed!");
    }
  } catch (error) {
    console.error("Error authorizing terminal:", error);
  }
}

module.exports = async function (callback) {
  await authorizeTerminal();
  callback();
};
