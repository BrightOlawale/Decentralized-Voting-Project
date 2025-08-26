package database

import (
	"database/sql"
	"fmt"
)

// RunMigrations executes database migrations
func RunMigrations(db *sql.DB) error {
	migrations := []string{
		createAuditLogsTable,
		createTerminalsTable,
		createVotersTable,
		createElectionsTable,
		createVotesTable,
		createPollingUnitsTable,
		createSystemLogsTable,
		createUsersTable,    // Added for API users
		createSessionsTable, // Added for session management
		createCandidatesTable,
		createIndices,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %v", i+1, err)
		}
	}

	return nil
}

// Database schema definitions
const createAuditLogsTable = `
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action VARCHAR(100) NOT NULL,
    user_id VARCHAR(255),
    polling_unit_id VARCHAR(50),
    details TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`

const createTerminalsTable = `
CREATE TABLE IF NOT EXISTS terminals (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    polling_unit_id VARCHAR(50),
    eth_address VARCHAR(42),
    public_key TEXT,
    status VARCHAR(20) DEFAULT 'registered',
    authorized BOOLEAN DEFAULT FALSE,
    last_heartbeat TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`

const createVotersTable = `
CREATE TABLE IF NOT EXISTS voters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nin VARCHAR(11) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    date_of_birth DATE,
    gender VARCHAR(10),
    polling_unit_id VARCHAR(50),
    fingerprint_hash VARCHAR(64) UNIQUE NOT NULL,
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);`

const createElectionsTable = `
CREATE TABLE IF NOT EXISTS elections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    blockchain_id VARCHAR(50),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    is_active BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`

const createVotesTable = `
CREATE TABLE IF NOT EXISTS votes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    blockchain_vote_id VARCHAR(50),
    verification_hash VARCHAR(64) UNIQUE NOT NULL,
    election_id INTEGER,
    polling_unit_id VARCHAR(50),
    candidate_id VARCHAR(50),
    encrypted_vote TEXT,
    transaction_hash VARCHAR(66),
    block_number INTEGER,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    synced_at TIMESTAMP,
    FOREIGN KEY (election_id) REFERENCES elections(id)
);`

const createCandidatesTable = `
CREATE TABLE IF NOT EXISTS candidates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    election_id INTEGER NOT NULL,
    candidate_id VARCHAR(100) NOT NULL,
    name VARCHAR(255),
    party VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(election_id, candidate_id),
    FOREIGN KEY (election_id) REFERENCES elections(id)
);`

const createPollingUnitsTable = `
CREATE TABLE IF NOT EXISTS polling_units (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    ward VARCHAR(100),
    lga VARCHAR(100),
    state VARCHAR(50),
    total_registered_voters INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`

const createSystemLogsTable = `
CREATE TABLE IF NOT EXISTS system_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    level VARCHAR(10) NOT NULL,
    message TEXT NOT NULL,
    component VARCHAR(50),
    details TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`

const createIndices = `
-- Create all indices separately (SQLite compatible)
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_terminals_polling_unit ON terminals(polling_unit_id);
CREATE INDEX IF NOT EXISTS idx_terminals_status ON terminals(status);
CREATE INDEX IF NOT EXISTS idx_terminals_authorized ON terminals(authorized);
CREATE INDEX IF NOT EXISTS idx_voters_nin ON voters(nin);
CREATE INDEX IF NOT EXISTS idx_voters_fingerprint ON voters(fingerprint_hash);
CREATE INDEX IF NOT EXISTS idx_voters_polling_unit ON voters(polling_unit_id);
CREATE INDEX IF NOT EXISTS idx_voters_active ON voters(is_active);
CREATE INDEX IF NOT EXISTS idx_elections_blockchain_id ON elections(blockchain_id);
CREATE INDEX IF NOT EXISTS idx_elections_active ON elections(is_active);
CREATE INDEX IF NOT EXISTS idx_elections_dates ON elections(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_votes_verification_hash ON votes(verification_hash);
CREATE INDEX IF NOT EXISTS idx_votes_election_id ON votes(election_id);
CREATE INDEX IF NOT EXISTS idx_votes_polling_unit ON votes(polling_unit_id);
CREATE INDEX IF NOT EXISTS idx_votes_status ON votes(status);
CREATE INDEX IF NOT EXISTS idx_votes_tx_hash ON votes(transaction_hash);
CREATE INDEX IF NOT EXISTS idx_polling_units_lga ON polling_units(lga);
CREATE INDEX IF NOT EXISTS idx_polling_units_state ON polling_units(state);
CREATE INDEX IF NOT EXISTS idx_polling_units_active ON polling_units(is_active);
CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs(level);
CREATE INDEX IF NOT EXISTS idx_system_logs_component ON system_logs(component);
CREATE INDEX IF NOT EXISTS idx_system_logs_created_at ON system_logs(created_at);
-- Additional performance indices
CREATE INDEX IF NOT EXISTS idx_audit_logs_composite ON audit_logs(action, created_at);
CREATE INDEX IF NOT EXISTS idx_votes_composite ON votes(election_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_terminals_heartbeat ON terminals(last_heartbeat);
CREATE INDEX IF NOT EXISTS idx_candidates_election ON candidates(election_id);
CREATE INDEX IF NOT EXISTS idx_candidates_candidate_id ON candidates(candidate_id);
`

// New tables for API functionality
const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(20) DEFAULT 'operator',
    permissions TEXT, -- JSON array of permissions
    is_active BOOLEAN DEFAULT TRUE,
    last_login TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`

const createSessionsTable = `
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id INTEGER NOT NULL,
    data TEXT, -- JSON session data
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);`
