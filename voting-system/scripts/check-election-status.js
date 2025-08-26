const SecureVotingSystem = artifacts.require("SecureVotingSystem");

module.exports = async function (callback) {
  try {
    const contract = await SecureVotingSystem.deployed();

    const totalElections = (await contract.getTotalElections()).toNumber();
    console.log("Total elections:", totalElections);

    const currentId = (await contract.getCurrentElectionId()).toNumber();
    console.log("Current active election:", currentId);

    if (totalElections === 0) {
      console.log("No elections yet.");
      return callback();
    }

    // Show details for the latest election
    const eid = totalElections;
    const details = await contract.getElectionDetails(eid);
    const name = details.name;
    const start = details.startTime;
    const end = details.endTime;
    const isActive = details.isActive;
    const candidates = details.candidates;
    const totalVotes = details.totalVotes;
    console.log({ eid, name, start, end, isActive, candidates, totalVotes });

    // Candidate-wise totals (using the new view)
    if (contract.getElectionCandidateResults) {
      const res = await contract.getElectionCandidateResults(eid);
      const ids = res.candidateIds || res[0];
      const countsRaw = res.voteCounts || res[1];
      const counts = countsRaw.map((x) => x.toString());
      console.log("Candidate results:");
      ids.forEach((id, i) => console.log(` - ${id}: ${counts[i]}`));
    } else {
      console.log(
        "getElectionCandidateResults not available; falling back per candidate loop..."
      );
      for (const c of candidates) {
        const n = (await contract.getElectionResults(eid, c)).toString();
        console.log(` - ${c}: ${n}`);
      }
    }

    callback();
  } catch (err) {
    console.error(err);
    callback(err);
  }
};
