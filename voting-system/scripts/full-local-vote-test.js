const SecureVotingSystem = artifacts.require("SecureVotingSystem");

/**
 * End-to-end local voting test using Truffle exec.
 *
 * What it does now:
 * 1) Ensures multiple terminals are authorized and a test polling unit exists.
 * 2) Ensures there is an active election (creates one that starts immediately if needed).
 * 3) Casts 100 randomized votes across all candidates from authorized terminals.
 * 4) Prints per-candidate totals at the end using getElectionCandidateResults.
 *
 * How to run (in project root):
 *   npx truffle exec scripts/full-local-vote-test.js --network development
 */
module.exports = async function (callback) {
  try {
    const contract = await SecureVotingSystem.deployed();
    const accounts = await web3.eth.getAccounts();

    const owner = accounts[0]; // Deployer/owner (admin actions)
    const terminalPool = accounts.slice(1, 5); // A few terminals for realism

    // Color helpers (ANSI)
    const c = {
      reset: "\x1b[0m",
      bold: "\x1b[1m",
      dim: "\x1b[2m",
      red: "\x1b[31m",
      green: "\x1b[32m",
      yellow: "\x1b[33m",
      blue: "\x1b[34m",
      magenta: "\x1b[35m",
      cyan: "\x1b[36m",
      gray: "\x1b[90m",
    };
    const step = (t) => console.log(`\n${c.cyan}${c.bold}▶ ${t}${c.reset}`);
    const info = (t, v) => console.log(`${c.gray}${t}:${c.reset} ${v}`);
    const ok = (t) => console.log(`${c.green}✓ ${t}${c.reset}`);
    const warn = (t) => console.log(`${c.yellow}! ${t}${c.reset}`);

    console.log(
      `\n${c.bold}${c.cyan}=== E2E LOCAL VOTING TEST (100 RANDOM VOTES) ===${c.reset}`
    );
    info("Contract address", contract.address);
    info("Owner", owner);
    info("Terminals", terminalPool.join(", "));

    // 1) Ensure terminals are authorized
    step("Authorizing terminals (if needed)");
    for (const t of terminalPool) {
      const okAuth = await contract.isTerminalAuthorized(t);
      if (!okAuth) {
        info("Authorizing", t);
        await contract.authorizeTerminal(t, true, { from: owner });
        ok(`Terminal authorized: ${t}`);
      } else {
        ok(`Already authorized: ${t}`);
      }
    }

    // 2) Ensure a test polling unit exists
    step("Ensuring test polling unit exists");
    const testPollingUnit = {
      id: "TEST_PU1",
      name: "Test PU 1",
      location: "Localhost",
      totalVoters: 100000,
    };
    try {
      info(
        "Registering polling unit",
        `${testPollingUnit.id} (${testPollingUnit.name})`
      );
      await contract.registerPollingUnit(
        testPollingUnit.id,
        testPollingUnit.name,
        testPollingUnit.location,
        testPollingUnit.totalVoters,
        { from: owner }
      );
      ok("Polling unit registered");
    } catch (_) {
      warn("Polling unit may already exist; continuing");
    }

    // 3) Ensure there is an active election
    step("Preparing active election");
    let currentElectionId = (await contract.getCurrentElectionId()).toNumber();
    if (currentElectionId === 0) {
      warn("No active election detected");
      const now = Math.floor(Date.now() / 1000);
      const startTime = now + 2; // Starts in ~2 seconds
      const endTime = now + 3600; // Ends in 1 hour
      const bootstrapCandidates = [
        "CANDIDATE_001",
        "CANDIDATE_002",
        "CANDIDATE_003",
      ];
      info(
        "Creating election",
        `start=${startTime}, end=${endTime}, candidates=${bootstrapCandidates.join(
          ", "
        )}`
      );
      await contract.createElection(
        "Local E2E Test",
        startTime,
        endTime,
        bootstrapCandidates,
        { from: owner }
      );
      const newElectionId = (await contract.getTotalElections()).toNumber();

      // Fast-forward Ganache time to startTime if needed
      const waitSeconds = startTime - Math.floor(Date.now() / 1000);
      if (waitSeconds > 0) {
        info("Fast-forward chain", `${waitSeconds + 1}s`);
        await new Promise((resolve) =>
          web3.currentProvider.send(
            {
              jsonrpc: "2.0",
              method: "evm_increaseTime",
              params: [waitSeconds + 1],
              id: Date.now(),
            },
            () => resolve()
          )
        );
        await new Promise((resolve) =>
          web3.currentProvider.send(
            {
              jsonrpc: "2.0",
              method: "evm_mine",
              params: [],
              id: Date.now() + 1,
            },
            () => resolve()
          )
        );
      }

      info("Starting election", newElectionId);
      await contract.startElection(newElectionId, { from: owner });
      currentElectionId = newElectionId;
      ok(`Active election ID: ${currentElectionId}`);
    } else {
      ok(`Using current active election: ${currentElectionId}`);
    }

    // Fetch the candidate list for the active election
    const det = await contract.getElectionDetails(currentElectionId);
    const candidates = det.candidates;
    info("Candidates", candidates.join(", "));

    // 4) Cast 100 randomized votes
    step("Casting randomized votes");
    const pollingUnitId = testPollingUnit.id;
    const totalVotesToCast = 100;
    info("Total votes to cast", totalVotesToCast);

    function randInt(n) {
      return Math.floor(Math.random() * n);
    }

    // In-memory tally for live progress
    const tally = {};
    for (const cid of candidates) tally[cid] = 0;

    for (let i = 0; i < totalVotesToCast; i++) {
      const term = terminalPool[i % terminalPool.length];
      const candidateId = candidates[randInt(candidates.length)];
      const verificationHash = web3.utils.keccak256(
        `voter-${currentElectionId}-${i}-${Math.random()}`
      );
      const encryptedVote = web3.utils.keccak256(
        `ciphertext-${i}-${Date.now()}`
      );

      await contract.castVote(
        verificationHash,
        encryptedVote,
        pollingUnitId,
        candidateId,
        { from: term }
      );

      tally[candidateId]++;
      if ((i + 1) % 10 === 0) {
        console.log(
          `${c.blue}${c.bold}Progress${c.reset} ${
            i + 1
          }/${totalVotesToCast} votes:`
        );
        for (const cid of candidates) {
          console.log(`  ${c.gray}- ${cid}:${c.reset} ${tally[cid]}`);
        }
      }
    }

    // Print per-candidate totals using the helper view, with safe fallback
    step("Final per-candidate totals (on-chain)");
    let printed = false;
    try {
      if (contract.getElectionCandidateResults) {
        const res = await contract.getElectionCandidateResults(
          currentElectionId
        );
        const ids = res.candidateIds || res[0];
        const countsRaw = res.voteCounts || res[1];
        const counts = countsRaw.map((x) => Number(x.toString()));
        const max = counts.length ? Math.max(...counts) : 0;
        const bar = (n) => {
          if (max === 0) return "";
          const width = 30;
          const len = Math.round((n / max) * width);
          return `${c.green}${"■".repeat(len)}${c.reset}${" ".repeat(
            Math.max(0, width - len)
          )}`;
        };
        ids.forEach((id, idx) =>
          console.log(
            ` - ${c.bold}${id}${c.reset}: ${counts[idx]
              .toString()
              .padStart(3, " ")} ${bar(counts[idx])}`
          )
        );
        printed = true;
      }
    } catch (e) {
      warn(
        "getElectionCandidateResults reverted (contract on-chain may be older). Falling back per-candidate..."
      );
    }

    if (!printed) {
      for (const cid of candidates) {
        const n = await contract.getElectionResults(currentElectionId, cid);
        console.log(` - ${c.bold}${cid}${c.reset}: ${n.toString()}`);
      }
    }

    console.log(
      `\n${c.bold}${c.green}✅ Randomized voting test completed.${c.reset}`
    );
    callback();
  } catch (err) {
    console.error("❌ E2E test failed:", err);
    callback(err);
  }
};
