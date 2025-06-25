const SecureVotingSystem = artifacts.require("SecureVotingSystem");

module.exports = async function (deployer, network, accounts) {
  console.log("🚀 Starting SecureVotingSystem deployment...");
  console.log("Network:", network);
  console.log("Deployer account:", accounts[0]);

  try {
    await deployer.deploy(SecureVotingSystem);
    const instance = await SecureVotingSystem.deployed();
    
    console.log("✅ SecureVotingSystem deployed successfully!");
    console.log("📍 Contract address:", instance.address);
    
    if (network === "development") {
      console.log("Setting up development environment...");
      
      // Authorize terminals
      for (let i = 1; i <= 3; i++) {
        if (accounts[i]) {
          await instance.authorizeTerminal(accounts[i], true);
          console.log(`Authorized terminal: ${accounts[i]}`);
        }
      }
      
      // Register polling units
      const pollingUnits = [
        { id: "PU001", name: "Community Hall A", location: "Lagos State", totalVoters: 500 },
        { id: "PU002", name: "Primary School B", location: "Ogun State", totalVoters: 750 },
        { id: "PU003", name: "Town Hall C", location: "Oyo State", totalVoters: 600 }
      ];
      
      for (const pu of pollingUnits) {
        await instance.registerPollingUnit(pu.id, pu.name, pu.location, pu.totalVoters);
        console.log(`Registered polling unit: ${pu.name} (${pu.id})`);
      }
      
      // Create election with future start time
      const candidates = ["CANDIDATE_001", "CANDIDATE_002", "CANDIDATE_003"];
      const startTime = Math.floor(Date.now() / 1000) + 300; // 5 minutes from now
      const endTime = startTime + (24 * 60 * 60); // 24 hours later
      
      await instance.createElection("2024 Presidential Election", startTime, endTime, candidates);
      console.log("Sample election created with future start time");
    }
    
  } catch (error) {
    console.error("❌ Deployment failed:", error);
    throw error;
  }
};
