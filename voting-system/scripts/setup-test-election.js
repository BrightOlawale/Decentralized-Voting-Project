const Web3 = require("web3");

// Configuration
const CONTRACT_ADDRESS = "0x345cA3e014Aaf5dcA488057592ee47305D9B3e10";
const OWNER_ADDRESS = "0x627306090abab3a6e1400e9345bc60c78a8bef57"; // First Ganache account

// Connect to local blockchain
const web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:8545"));

// Contract ABI (minimal for election functions)
const contractABI = [
  {
    inputs: [
      {
        internalType: "string",
        name: "_name",
        type: "string",
      },
      {
        internalType: "uint256",
        name: "_startTime",
        type: "uint256",
      },
      {
        internalType: "uint256",
        name: "_endTime",
        type: "uint256",
      },
      {
        internalType: "string[]",
        name: "_candidates",
        type: "string[]",
      },
    ],
    name: "createElection",
    outputs: [
      {
        internalType: "uint256",
        name: "",
        type: "uint256",
      },
    ],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "uint256",
        name: "_electionId",
        type: "uint256",
      },
    ],
    name: "startElection",
    outputs: [],
    stateMutability: "nonpayable",
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
  {
    inputs: [],
    name: "getTotalElections",
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
  {
    inputs: [
      {
        internalType: "uint256",
        name: "_electionId",
        type: "uint256",
      },
    ],
    name: "getElectionDetails",
    outputs: [
      {
        internalType: "string",
        name: "name",
        type: "string",
      },
      {
        internalType: "uint256",
        name: "startTime",
        type: "uint256",
      },
      {
        internalType: "uint256",
        name: "endTime",
        type: "uint256",
      },
      {
        internalType: "bool",
        name: "isActive",
        type: "bool",
      },
      {
        internalType: "string[]",
        name: "candidates",
        type: "string[]",
      },
      {
        internalType: "uint256",
        name: "totalVotes",
        type: "uint256",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
];

async function setupTestElection() {
  try {
    // Create contract instance
    const contract = new web3.eth.Contract(contractABI, CONTRACT_ADDRESS);

    // Check current election
    console.log("Checking current election...");
    const currentElectionId = await contract.methods
      .getCurrentElectionId()
      .call();
    console.log(`Current election ID: ${currentElectionId}`);

    if (currentElectionId > 0) {
      console.log("An election is already active!");
      const electionDetails = await contract.methods
        .getElectionDetails(currentElectionId)
        .call();
      console.log("Election details:", electionDetails);
      return currentElectionId;
    }

    // Check total elections
    const totalElections = await contract.methods.getTotalElections().call();
    console.log(`Total elections created: ${totalElections}`);

    // Get the owner account
    const accounts = await web3.eth.getAccounts();
    const ownerAccount = accounts[0];

    console.log(`Using owner account: ${ownerAccount}`);

    // Create test election with start time in the future
    const now = Math.floor(Date.now() / 1000);
    const startTime = now + 60; // Start in 1 minute
    const endTime = now + 3600; // End in 1 hour

    const testCandidates = ["CANDIDATE_A", "CANDIDATE_B", "CANDIDATE_C"];

    console.log("Creating test election...");
    console.log(`Start time: ${new Date(startTime * 1000)}`);
    console.log(`End time: ${new Date(endTime * 1000)}`);
    console.log(`Candidates: ${testCandidates.join(", ")}`);

    // Create election transaction
    const createData = contract.methods
      .createElection("Test Election 2024", startTime, endTime, testCandidates)
      .encodeABI();

    const createTx = {
      from: ownerAccount,
      to: CONTRACT_ADDRESS,
      data: createData,
      gas: 500000,
      gasPrice: web3.utils.toWei("20", "gwei"),
    };

    // Estimate gas
    const createGasEstimate = await web3.eth.estimateGas(createTx);
    console.log(`Create election estimated gas: ${createGasEstimate}`);
    createTx.gas = createGasEstimate;

    // Send create transaction
    const createResult = await web3.eth.sendTransaction(createTx);
    console.log(
      `Create election transaction hash: ${createResult.transactionHash}`
    );

    // Wait a moment for the transaction to be processed
    await new Promise((resolve) => setTimeout(resolve, 2000));

    // Get the new election ID (should be totalElections + 1)
    const newTotalElections = await contract.methods.getTotalElections().call();
    console.log(`New total elections: ${newTotalElections}`);

    const newElectionId = newTotalElections;
    console.log(`New election ID: ${newElectionId}`);

    // Wait for the start time to be reached
    console.log("Waiting for election start time...");
    const waitTime = (startTime - Math.floor(Date.now() / 1000)) * 1000;
    if (waitTime > 0) {
      console.log(`Waiting ${Math.ceil(waitTime / 1000)} seconds...`);
      await new Promise((resolve) => setTimeout(resolve, waitTime + 1000)); // Add 1 second buffer
    }

    // Start the election
    console.log("Starting election...");
    const startData = contract.methods.startElection(newElectionId).encodeABI();

    const startTx = {
      from: ownerAccount,
      to: CONTRACT_ADDRESS,
      data: startData,
      gas: 200000,
      gasPrice: web3.utils.toWei("20", "gwei"),
    };

    // Estimate gas
    const startGasEstimate = await web3.eth.estimateGas(startTx);
    console.log(`Start election estimated gas: ${startGasEstimate}`);
    startTx.gas = startGasEstimate;

    // Send start transaction
    const startResult = await web3.eth.sendTransaction(startTx);
    console.log(
      `Start election transaction hash: ${startResult.transactionHash}`
    );

    // Wait a moment for the transaction to be processed
    await new Promise((resolve) => setTimeout(resolve, 2000));

    // Verify election is active
    const finalElectionId = await contract.methods
      .getCurrentElectionId()
      .call();
    console.log(`Final election ID: ${finalElectionId}`);

    if (finalElectionId > 0) {
      const finalDetails = await contract.methods
        .getElectionDetails(finalElectionId)
        .call();
      console.log("Final election details:", finalDetails);
      console.log("✅ Test election setup successful!");
      return finalElectionId;
    } else {
      console.log("❌ Election setup failed!");
      return null;
    }
  } catch (error) {
    console.error("Error setting up test election:", error);
    return null;
  }
}

// Run the setup
setupTestElection();
