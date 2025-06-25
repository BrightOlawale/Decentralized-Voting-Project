# Blockchain Secured Online Voting System with Biometric Verification

A secure, transparent, and tamper-proof voting system built with Go, Ethereum blockchain, and biometric verification.

## Features

- **Dual Authentication**: NIN/BVN + Fingerprint verification
- **Blockchain Security**: Immutable vote recording on Ethereum
- **Offline Capability**: Works without constant internet connection
- **Real-time Monitoring**: System health and vote tracking
- **Cross-platform**: Supports Linux, macOS, and Windows

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- PostgreSQL (for central server)
- SQLite (for terminals)

### Development Setup

1. Start the blockchain:
   ```bash
   make start-blockchain
   ```

2. Deploy smart contracts:
   ```bash
   make deploy-contracts
   ```

3. Run the terminal:
   ```bash
   make run-terminal
   ```

4. Run the central server:
   ```bash
   make run-server
   ```

### Building

```bash
make build
```

### Testing

```bash
make test
```

## Architecture

- **cmd/**: Application entry points
- **internal/**: Private application code
- **pkg/**: Public library code
- **contracts/**: Smart contracts and migrations
- **configs/**: Configuration files
- **web/**: Web interface assets

## Configuration

Configuration files are located in the `configs/` directory:
- `terminal.yaml`: Voting terminal configuration
- `server.yaml`: Central server configuration

## Security Features

- Multi-factor biometric authentication
- Encrypted vote storage
- Blockchain immutability
- Real-time anomaly detection
- Comprehensive audit trails

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License.
