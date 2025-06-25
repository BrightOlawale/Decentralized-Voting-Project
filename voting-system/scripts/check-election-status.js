const Web3 = require("web3");

// Configuration
const CONTRACT_ADDRESS = "0x345cA3e014Aaf5dcA488057592ee47305D9B3e10";

// Connect to local blockchain
const web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:8545"));

// Contract ABI (minimal for status functions)
const contractABI = [
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

async function checkElectionStatus() {
  try {
    // Create contract instance
    const contract = new web3.eth.Contract(contractABI, CONTRACT_ADDRESS);

    // Check current election
    console.log("Checking election status...");
    const currentElectionId = await contract.methods
      .getCurrentElectionId()
      .call();
    console.log(`Current election ID: ${currentElectionId}`);

    if (currentElectionId > 0) {
      const electionDetails = await contract.methods
        .getElectionDetails(currentElectionId)
        .call();
      console.log("Election details:");
      console.log(`  Name: ${electionDetails.name}`);
      console.log(
        `  Start time: ${new Date(electionDetails.startTime * 1000)}`
      );
      console.log(`  End time: ${new Date(electionDetails.endTime * 1000)}`);
      console.log(`  Is active: ${electionDetails.isActive}`);
      console.log(`  Candidates: ${electionDetails.candidates.join(", ")}`);
      console.log(`  Total votes: ${electionDetails.totalVotes}`);

      if (electionDetails.isActive) {
        console.log("✅ Election is active and ready for voting!");
      } else {
        console.log("⏳ Election is created but not yet active");
      }
    } else {
      console.log("❌ No active election found");
    }
  } catch (error) {
    console.error("Error checking election status:", error);
  }
}

// Run the check
checkElectionStatus();
