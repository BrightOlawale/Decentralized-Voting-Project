const Web3 = require("web3");

// Configuration
const CONTRACT_ADDRESS = "0x345cA3e014Aaf5dcA488057592ee47305D9B3e10";
const TEST_TERMINAL_ADDRESS = "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23";
const TEST_PRIVATE_KEY =
  "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318";
const FUNDING_AMOUNT = "0x8AC7230489E80000"; // 10 ETH in wei

// Connect to local blockchain
const web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:8545"));

// Contract ABI (minimal for authorization functions)
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

async function setupTestEnvironment() {
  try {
    console.log("🚀 Setting up test environment...\n");

    // Get accounts
    const accounts = await web3.eth.getAccounts();
    const ownerAccount = accounts[0]; // First account (has funds)

    console.log(`Owner account: ${ownerAccount}`);
    console.log(`Test terminal account: ${TEST_TERMINAL_ADDRESS}\n`);

    // Step 1: Check and fund the test account
    console.log("💰 Step 1: Checking test account balance...");
    const testBalance = await web3.eth.getBalance(TEST_TERMINAL_ADDRESS);
    console.log(
      `Current balance: ${web3.utils.fromWei(testBalance, "ether")} ETH`
    );

    if (web3.utils.toBN(testBalance).lt(web3.utils.toBN(FUNDING_AMOUNT))) {
      console.log("Funding test account...");

      const fundTx = {
        from: ownerAccount,
        to: TEST_TERMINAL_ADDRESS,
        value: FUNDING_AMOUNT,
        gas: 21000,
        gasPrice: web3.utils.toWei("20", "gwei"),
      };

      const fundResult = await web3.eth.sendTransaction(fundTx);
      console.log(`Funding transaction hash: ${fundResult.transactionHash}`);

      // Wait a moment for transaction to be processed
      await new Promise((resolve) => setTimeout(resolve, 2000));

      const newBalance = await web3.eth.getBalance(TEST_TERMINAL_ADDRESS);
      console.log(
        `New balance: ${web3.utils.fromWei(newBalance, "ether")} ETH ✅\n`
      );
    } else {
      console.log("Test account already has sufficient funds ✅\n");
    }

    // Step 2: Check and authorize the terminal
    console.log("🔐 Step 2: Checking terminal authorization...");
    const contract = new web3.eth.Contract(contractABI, CONTRACT_ADDRESS);

    const isAuthorized = await contract.methods
      .isTerminalAuthorized(TEST_TERMINAL_ADDRESS)
      .call();
    console.log(`Currently authorized: ${isAuthorized}`);

    if (!isAuthorized) {
      console.log("Authorizing terminal...");

      const authorizeData = contract.methods
        .authorizeTerminal(TEST_TERMINAL_ADDRESS, true)
        .encodeABI();

      const authTx = {
        from: ownerAccount,
        to: CONTRACT_ADDRESS,
        data: authorizeData,
        gas: 100000,
        gasPrice: web3.utils.toWei("20", "gwei"),
      };

      // Estimate gas
      const gasEstimate = await web3.eth.estimateGas(authTx);
      console.log(`Estimated gas: ${gasEstimate}`);
      authTx.gas = gasEstimate;

      const authResult = await web3.eth.sendTransaction(authTx);
      console.log(
        `Authorization transaction hash: ${authResult.transactionHash}`
      );

      // Wait a moment for transaction to be processed
      await new Promise((resolve) => setTimeout(resolve, 2000));

      const newAuthStatus = await contract.methods
        .isTerminalAuthorized(TEST_TERMINAL_ADDRESS)
        .call();
      console.log(`New authorization status: ${newAuthStatus} ✅\n`);
    } else {
      console.log("Terminal already authorized ✅\n");
    }

    // Step 3: Verify setup
    console.log("✅ Step 3: Verifying setup...");
    const finalBalance = await web3.eth.getBalance(TEST_TERMINAL_ADDRESS);
    const finalAuthStatus = await contract.methods
      .isTerminalAuthorized(TEST_TERMINAL_ADDRESS)
      .call();

    console.log(
      `Final test account balance: ${web3.utils.fromWei(
        finalBalance,
        "ether"
      )} ETH`
    );
    console.log(`Final terminal authorization: ${finalAuthStatus}`);

    if (
      web3.utils.toBN(finalBalance).gte(web3.utils.toBN(FUNDING_AMOUNT)) &&
      finalAuthStatus
    ) {
      console.log("\n🎉 Test environment setup completed successfully!");
      console.log("\nYou can now run your tests with:");
      console.log(
        `CONTRACT_ADDRESS="${CONTRACT_ADDRESS}" go test ./internal/blockchain -v`
      );
    } else {
      console.log("\n❌ Test environment setup failed!");
      if (!web3.utils.toBN(finalBalance).gte(web3.utils.toBN(FUNDING_AMOUNT))) {
        console.log("- Test account funding failed");
      }
      if (!finalAuthStatus) {
        console.log("- Terminal authorization failed");
      }
    }
  } catch (error) {
    console.error("❌ Error setting up test environment:", error);
  }
}

// Run the setup
setupTestEnvironment();
