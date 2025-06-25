// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/utils/Counters.sol";

/**
 * @title SecureVotingSystem
 * @dev A secure blockchain-based voting system with biometric verification
 * @author Olawale Olatunji Bright
 */
contract SecureVotingSystem is Ownable, ReentrancyGuard {
    using Counters for Counters.Counter;
    
    // State variables
    Counters.Counter private _voteCounter;
    Counters.Counter private _electionCounter;
    
    struct Vote {
        bytes32 verificationHash;      // Hash of NIN + BVN + Fingerprint
        bytes32 encryptedVote;         // Encrypted vote data
        uint256 timestamp;             // When vote was cast
        string pollingUnitId;          // Where vote was cast
        uint256 electionId;            // Which election
        bool isValid;                  // Vote validity status
    }
    
    struct Election {
        uint256 id;
        string name;
        uint256 startTime;
        uint256 endTime;
        bool isActive;
        string[] candidates;
        mapping(string => uint256) candidateVotes; // Candidate -> vote count
        uint256 totalVotes;
    }
    
    struct Candidate {
        string id;
        string name;
        string party;
        uint256 voteCount;
    }
    
    struct PollingUnit {
        string id;
        string name;
        string location;
        uint256 totalVoters;
        uint256 votesRecorded;
        bool isActive;
    }
    
    // Mappings
    mapping(bytes32 => bool) public hasVoted;                    // verificationHash -> voted status
    mapping(uint256 => Vote) public votes;                      // voteId -> Vote
    mapping(bytes32 => uint256) public verificationHashToVoteId; // verificationHash -> voteId
    mapping(uint256 => Election) public elections;              // electionId -> Election
    mapping(address => bool) public authorizedTerminals;        // terminal addresses
    mapping(string => PollingUnit) public pollingUnits;         // pollingUnitId -> PollingUnit
    mapping(uint256 => mapping(string => uint256)) public electionResults; // electionId -> candidateId -> votes
    
    // Current active election
    uint256 public currentElectionId;
    
    // Events
    event VoteCast(
        bytes32 indexed verificationHash,
        string indexed pollingUnitId,
        uint256 indexed electionId,
        uint256 timestamp,
        uint256 voteId
    );
    
    event ElectionCreated(
        uint256 indexed electionId,
        string name,
        uint256 startTime,
        uint256 endTime
    );
    
    event ElectionStarted(uint256 indexed electionId, uint256 timestamp);
    event ElectionEnded(uint256 indexed electionId, uint256 timestamp);
    
    event TerminalAuthorized(address indexed terminal, bool status);
    event PollingUnitRegistered(string indexed pollingUnitId, string name);
    event VoteInvalidated(uint256 indexed voteId, string reason);
    
    // Modifiers
    modifier onlyAuthorizedTerminal() {
        require(authorizedTerminals[msg.sender], "VotingSystem: Unauthorized terminal");
        _;
    }
    
    modifier onlyDuringElection() {
        require(currentElectionId > 0, "VotingSystem: No active election");
        Election storage election = elections[currentElectionId];
        require(election.isActive, "VotingSystem: Election not active");
        require(
            block.timestamp >= election.startTime && 
            block.timestamp <= election.endTime,
            "VotingSystem: Election not in session"
        );
        _;
    }
    
    modifier onlyAfterElection(uint256 _electionId) {
        Election storage election = elections[_electionId];
        require(
            !election.isActive || block.timestamp > election.endTime,
            "VotingSystem: Election still in progress"
        );
        _;
    }
    
    modifier validPollingUnit(string memory _pollingUnitId) {
        require(pollingUnits[_pollingUnitId].isActive, "VotingSystem: Invalid polling unit");
        _;
    }
    
    constructor() {
        // Initialize with deployer as first authorized terminal
        authorizedTerminals[msg.sender] = true;
        emit TerminalAuthorized(msg.sender, true);
    }
    
    // Election Management Functions
    
    /**
     * @dev Create a new election
     * @param _name Election name
     * @param _startTime Election start timestamp
     * @param _endTime Election end timestamp
     * @param _candidates Array of candidate IDs
     */
    function createElection(
        string memory _name,
        uint256 _startTime,
        uint256 _endTime,
        string[] memory _candidates
    ) external onlyOwner returns (uint256) {
        require(_startTime > block.timestamp, "VotingSystem: Start time must be in future");
        require(_endTime > _startTime, "VotingSystem: End time must be after start time");
        require(_candidates.length > 0, "VotingSystem: Must have candidates");
        
        _electionCounter.increment();
        uint256 electionId = _electionCounter.current();
        
        Election storage newElection = elections[electionId];
        newElection.id = electionId;
        newElection.name = _name;
        newElection.startTime = _startTime;
        newElection.endTime = _endTime;
        newElection.isActive = false;
        newElection.candidates = _candidates;
        newElection.totalVotes = 0;
        
        // Initialize candidate vote counts
        for (uint i = 0; i < _candidates.length; i++) {
            newElection.candidateVotes[_candidates[i]] = 0;
        }
        
        emit ElectionCreated(electionId, _name, _startTime, _endTime);
        return electionId;
    }
    
    /**
     * @dev Start an election
     * @param _electionId Election ID to start
     */
    function startElection(uint256 _electionId) external onlyOwner {
        require(_electionId > 0 && _electionId <= _electionCounter.current(), "VotingSystem: Invalid election ID");
        require(currentElectionId == 0, "VotingSystem: Another election is active");
        
        Election storage election = elections[_electionId];
        require(!election.isActive, "VotingSystem: Election already started");
        require(block.timestamp >= election.startTime, "VotingSystem: Election start time not reached");
        require(block.timestamp < election.endTime, "VotingSystem: Election has expired");
        
        election.isActive = true;
        currentElectionId = _electionId;
        
        emit ElectionStarted(_electionId, block.timestamp);
    }
    
    /**
     * @dev End the current active election
     */
    function endElection() external onlyOwner {
        require(currentElectionId > 0, "VotingSystem: No active election");
        
        Election storage election = elections[currentElectionId];
        require(election.isActive, "VotingSystem: Election not active");
        
        election.isActive = false;
        uint256 endedElectionId = currentElectionId;
        currentElectionId = 0;
        
        emit ElectionEnded(endedElectionId, block.timestamp);
    }
    
    // Voting Functions
    
    /**
     * @dev Cast a vote in the current election
     * @param _verificationHash Hash combining NIN, BVN, and biometric data
     * @param _encryptedVote Encrypted vote data
     * @param _pollingUnitId Polling unit where vote is cast
     * @param _candidateId ID of the candidate being voted for
     */
    function castVote(
        bytes32 _verificationHash,
        bytes32 _encryptedVote,
        string memory _pollingUnitId,
        string memory _candidateId
    ) external 
        onlyAuthorizedTerminal 
        onlyDuringElection 
        validPollingUnit(_pollingUnitId)
        nonReentrant 
        returns (uint256) {
        
        // Check if voter has already voted in this election
        require(!hasVoted[_verificationHash], "VotingSystem: Voter has already cast a vote");
        
        // Validate candidate
        Election storage election = elections[currentElectionId];
        bool validCandidate = false;
        for (uint i = 0; i < election.candidates.length; i++) {
            if (keccak256(bytes(election.candidates[i])) == keccak256(bytes(_candidateId))) {
                validCandidate = true;
                break;
            }
        }
        require(validCandidate, "VotingSystem: Invalid candidate");
        
        // Mark as voted
        hasVoted[_verificationHash] = true;
        
        // Increment vote counter
        _voteCounter.increment();
        uint256 voteId = _voteCounter.current();
        
        // Store vote
        votes[voteId] = Vote({
            verificationHash: _verificationHash,
            encryptedVote: _encryptedVote,
            timestamp: block.timestamp,
            pollingUnitId: _pollingUnitId,
            electionId: currentElectionId,
            isValid: true
        });
        
        // Map verification hash to vote ID
        verificationHashToVoteId[_verificationHash] = voteId;
        
        // Update election tallies
        election.candidateVotes[_candidateId]++;
        election.totalVotes++;
        electionResults[currentElectionId][_candidateId]++;
        
        // Update polling unit count
        pollingUnits[_pollingUnitId].votesRecorded++;
        
        // Emit event
        emit VoteCast(_verificationHash, _pollingUnitId, currentElectionId, block.timestamp, voteId);
        
        return voteId;
    }
    
    /**
     * @dev Check if a voter has already voted
     * @param _verificationHash Voter's verification hash
     * @return bool Whether the voter has voted
     */
    function hasVoterVoted(bytes32 _verificationHash) external view returns (bool) {
        return hasVoted[_verificationHash];
    }
    
    // Polling Unit Management
    
    /**
     * @dev Register a new polling unit
     * @param _pollingUnitId Unique polling unit ID
     * @param _name Polling unit name
     * @param _location Polling unit location
     * @param _totalVoters Expected number of voters
     */
    function registerPollingUnit(
        string memory _pollingUnitId,
        string memory _name,
        string memory _location,
        uint256 _totalVoters
    ) external onlyOwner {
        require(bytes(_pollingUnitId).length > 0, "VotingSystem: Invalid polling unit ID");
        require(!pollingUnits[_pollingUnitId].isActive, "VotingSystem: Polling unit already exists");
        
        pollingUnits[_pollingUnitId] = PollingUnit({
            id: _pollingUnitId,
            name: _name,
            location: _location,
            totalVoters: _totalVoters,
            votesRecorded: 0,
            isActive: true
        });
        
        emit PollingUnitRegistered(_pollingUnitId, _name);
    }
    
    // Terminal Management
    
    /**
     * @dev Authorize or deauthorize a terminal
     * @param _terminal Terminal address
     * @param _status Authorization status
     */
    function authorizeTerminal(address _terminal, bool _status) external onlyOwner {
        require(_terminal != address(0), "VotingSystem: Invalid terminal address");
        authorizedTerminals[_terminal] = _status;
        emit TerminalAuthorized(_terminal, _status);
    }
    
    /**
     * @dev Check if a terminal is authorized
     * @param _terminal Terminal address to check
     * @return bool Authorization status
     */
    function isTerminalAuthorized(address _terminal) external view returns (bool) {
        return authorizedTerminals[_terminal];
    }
    
    // Query Functions
    
    /**
     * @dev Get vote details by vote ID
     * @param _voteId Vote ID
     * @return verificationHash Hash of NIN + BVN + Fingerprint
     * @return encryptedVote Encrypted vote data
     * @return timestamp When vote was cast
     * @return pollingUnitId Where vote was cast
     * @return electionId Which election
     * @return isValid Vote validity status
     */
    function getVoteDetails(uint256 _voteId) external view returns (
        bytes32 verificationHash,
        bytes32 encryptedVote,
        uint256 timestamp,
        string memory pollingUnitId,
        uint256 electionId,
        bool isValid
    ) {
        require(_voteId > 0 && _voteId <= _voteCounter.current(), "VotingSystem: Invalid vote ID");
        Vote memory vote = votes[_voteId];
        return (
            vote.verificationHash,
            vote.encryptedVote,
            vote.timestamp,
            vote.pollingUnitId,
            vote.electionId,
            vote.isValid
        );
    }
    
    /**
     * @dev Get election details
     * @param _electionId Election ID
     * @return name Election name
     * @return startTime Election start timestamp
     * @return endTime Election end timestamp
     * @return isActive Whether election is active
     * @return candidates Array of candidate IDs
     * @return totalVotes Total votes cast
     */
    function getElectionDetails(uint256 _electionId) external view returns (
        string memory name,
        uint256 startTime,
        uint256 endTime,
        bool isActive,
        string[] memory candidates,
        uint256 totalVotes
    ) {
        require(_electionId > 0 && _electionId <= _electionCounter.current(), "VotingSystem: Invalid election ID");
        Election storage election = elections[_electionId];
        return (
            election.name,
            election.startTime,
            election.endTime,
            election.isActive,
            election.candidates,
            election.totalVotes
        );
    }
    
    /**
     * @dev Get election results
     * @param _electionId Election ID
     * @param _candidateId Candidate ID
     * @return uint256 Vote count for the candidate
     */
    function getElectionResults(uint256 _electionId, string memory _candidateId) 
        external view returns (uint256) {
        return electionResults[_electionId][_candidateId];
    }
    
    /**
     * @dev Get current active election ID
     * @return uint256 Current election ID (0 if none active)
     */
    function getCurrentElectionId() external view returns (uint256) {
        return currentElectionId;
    }
    
    /**
     * @dev Get total number of votes cast
     * @return uint256 Total vote count
     */
    function getTotalVotes() external view returns (uint256) {
        return _voteCounter.current();
    }
    
    /**
     * @dev Get total number of elections created
     * @return uint256 Total election count
     */
    function getTotalElections() external view returns (uint256) {
        return _electionCounter.current();
    }
    
    /**
     * @dev Get polling unit vote count
     * @param _pollingUnitId Polling unit ID
     * @return uint256 Number of votes recorded at the polling unit
     */
    function getPollingUnitVoteCount(string memory _pollingUnitId) 
        external view returns (uint256) {
        return pollingUnits[_pollingUnitId].votesRecorded;
    }
    
    // Emergency Functions
    
    /**
     * @dev Emergency pause of current election
     */
    function emergencyPause() external onlyOwner {
        if (currentElectionId > 0) {
            elections[currentElectionId].isActive = false;
        }
    }
    
    /**
     * @dev Invalidate a specific vote
     * @param _voteId Vote ID to invalidate
     * @param _reason Reason for invalidation
     */
    function invalidateVote(uint256 _voteId, string memory _reason) external onlyOwner {
        require(_voteId > 0 && _voteId <= _voteCounter.current(), "VotingSystem: Invalid vote ID");
        require(votes[_voteId].isValid, "VotingSystem: Vote already invalid");
        
        votes[_voteId].isValid = false;
        
        // Update election tallies if election is still active
        Vote memory vote = votes[_voteId];
        if (elections[vote.electionId].isActive) {
            elections[vote.electionId].totalVotes--;
            pollingUnits[vote.pollingUnitId].votesRecorded--;
        }
        
        emit VoteInvalidated(_voteId, _reason);
    }
    
    // Audit Functions
    
    /**
     * @dev Get votes cast in a time range
     * @param _startTime Start timestamp
     * @param _endTime End timestamp
     * @return uint256[] Array of vote IDs
     */
    function getVotesByTimeRange(uint256 _startTime, uint256 _endTime) 
        external view returns (uint256[] memory) {
        require(_endTime >= _startTime, "VotingSystem: Invalid time range");
        
        uint256 totalVotes = _voteCounter.current();
        uint256[] memory tempIds = new uint256[](totalVotes);
        uint256 count = 0;
        
        for (uint256 i = 1; i <= totalVotes; i++) {
            if (votes[i].timestamp >= _startTime && votes[i].timestamp <= _endTime) {
                tempIds[count] = i;
                count++;
            }
        }
        
        // Create properly sized array
        uint256[] memory voteIds = new uint256[](count);
        for (uint256 i = 0; i < count; i++) {
            voteIds[i] = tempIds[i];
        }
        
        return voteIds;
    }
    
    /**
     * @dev Get comprehensive election statistics
     * @param _electionId Election ID
     * @return totalVotes Total votes cast
     * @return validVotes Number of valid votes
     * @return invalidVotes Number of invalid votes
     * @return duration Election duration in seconds
     * @return isCompleted Whether election is completed
     */
    function getElectionStatistics(uint256 _electionId) external view returns (
        uint256 totalVotes,
        uint256 validVotes,
        uint256 invalidVotes,
        uint256 duration,
        bool isCompleted
    ) {
        require(_electionId > 0 && _electionId <= _electionCounter.current(), "VotingSystem: Invalid election ID");
        
        Election storage election = elections[_electionId];
        uint256 total = election.totalVotes;
        uint256 valid = 0;
        uint256 invalid = 0;
        
        // Count valid vs invalid votes for this election
        for (uint256 i = 1; i <= _voteCounter.current(); i++) {
            if (votes[i].electionId == _electionId) {
                if (votes[i].isValid) {
                    valid++;
                } else {
                    invalid++;
                }
            }
        }
        
        uint256 electionDuration = election.endTime - election.startTime;
        bool completed = !election.isActive && block.timestamp > election.endTime;
        
        return (total, valid, invalid, electionDuration, completed);
    }
}
