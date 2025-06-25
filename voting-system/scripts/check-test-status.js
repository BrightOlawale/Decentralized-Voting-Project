const Web3 = require("web3");

// Configuration
const CONTRACT_ADDRESS = "0x345cA3e014Aaf5dcA488057592ee47305D9B3e10";
const TEST_TERMINAL_ADDRESS = "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23";

// Connect to local blockchain
const web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:8545"));

// Contract ABI (minimal for status functions)
const contractABI = [
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
  {
    inputs: [],
    name: "getCurrentElectionId",
    outputs: [
      {
        internalType: "uint256",
        name: "",
        type: "uint256",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
];

async function checkTestStatus() {
  try {
    console.log("🔍 Checking test environment status...\n");

    // Check blockchain connection
    console.log("📡 Blockchain Connection:");
    const blockNumber = await web3.eth.getBlockNumber();
    console.log(`  Current block: ${blockNumber}`);

    // Check test account balance
    console.log("\n💰 Test Account Balance:");
    const balance = await web3.eth.getBalance(TEST_TERMINAL_ADDRESS);
    const balanceEth = web3.utils.fromWei(balance, "ether");
    console.log(`  Address: ${TEST_TERMINAL_ADDRESS}`);
    console.log(`  Balance: ${balanceEth} ETH`);

    // Check terminal authorization
    console.log("\n🔐 Terminal Authorization:");
    const contract = new web3.eth.Contract(contractABI, CONTRACT_ADDRESS);
    const isAuthorized = await contract.methods
      .isTerminalAuthorized(TEST_TERMINAL_ADDRESS)
      .call();
    console.log(`  Authorized: ${isAuthorized ? "✅ Yes" : "❌ No"}`);

    // Check election status
    console.log("\n🗳️  Election Status:");
    const electionId = await contract.methods.getCurrentElectionId().call();
    console.log(`  Current election ID: ${electionId}`);

    // Summary
    console.log("\n📊 Summary:");
    const hasFunds = web3.utils.toBN(balance).gt(web3.utils.toBN("0"));
    const hasElection = electionId > 0;

    console.log(`  ✅ Blockchain connected: ${blockNumber > 0 ? "Yes" : "No"}`);
    console.log(`  ✅ Test account funded: ${hasFunds ? "Yes" : "No"}`);
    console.log(`  ✅ Terminal authorized: ${isAuthorized ? "Yes" : "No"}`);
    console.log(`  ✅ Active election: ${hasElection ? "Yes" : "No"}`);

    const allReady = blockNumber > 0 && hasFunds && isAuthorized && hasElection;

    if (allReady) {
      console.log("\n🎉 Test environment is ready!");
      console.log("\nRun tests with:");
      console.log(
        `CONTRACT_ADDRESS="${CONTRACT_ADDRESS}" go test ./internal/blockchain -v`
      );
    } else {
      console.log("\n⚠️  Test environment needs setup!");
      console.log("\nRun setup with:");
      console.log("node scripts/setup-test-environment.js");
    }
  } catch (error) {
    console.error("❌ Error checking status:", error);
    console.log("\nMake sure Ganache is running on port 8545");
  }
}

// Run the check
checkTestStatus();
