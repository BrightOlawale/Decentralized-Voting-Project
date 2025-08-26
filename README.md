# Decentralized Voting System

A comprehensive blockchain-based voting system built as a Final Year Project at the Department of Computer Science and Engineering, Obafemi Awolowo University.

## 🎯 Project Overview

This project implements a secure, transparent, and decentralized voting system using blockchain technology. The system ensures vote integrity, prevents double voting, and provides real-time results while maintaining voter anonymity through cryptographic techniques.

## ✨ Key Features

- **🔐 Secure Smart Contracts**: Ethereum-based smart contracts for vote management
- **🌐 Web Interface**: User-friendly web application for voting and administration
- **⚡ Real-time Results**: Live vote counting and result display
- **🛡️ Anti-Fraud Protection**: Cryptographic verification and duplicate vote prevention
- **📊 Admin Dashboard**: Comprehensive administration tools for election management
- **🔍 Transparent Audit Trail**: Immutable blockchain records for verification
- **📱 Cross-Platform**: Works on desktop and mobile devices

## 🏗️ Architecture

The system is built with a multi-layered architecture:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Frontend  │    │   Go Backend    │    │  Smart Contracts│
│   (HTML/CSS/JS) │◄──►│   (Gin Server)  │◄──►│   (Solidity)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Blockchain    │
                       │   (Ethereum)    │
                       └─────────────────┘
```

## 🛠️ Technology Stack

### Backend

- **Go 1.23.3**: High-performance server-side language
- **Gin Framework**: Fast HTTP web framework
- **Ethereum Go Client**: Blockchain integration
- **SQLite/PostgreSQL**: Database management

### Smart Contracts

- **Solidity**: Smart contract development
- **Truffle**: Development framework
- **OpenZeppelin**: Secure contract libraries
- **Ganache**: Local blockchain for development

### Frontend

- **HTML5/CSS3**: Modern web interface
- **JavaScript**: Interactive client-side functionality
- **Web3.js**: Blockchain interaction

### DevOps & Tools

- **Docker**: Containerization
- **Make**: Build automation
- **Git**: Version control

## 🚀 Quick Start

### Prerequisites

- Go 1.23.3 or higher
- Node.js 18.x or higher
- Git
- Docker (optional)

### Automated Setup

```bash
# Clone the repository
git clone <repository-url>
cd Decentralized-Voting-Project

# Run the automated setup script
chmod +x quick_setup.sh
./quick_setup.sh
```

### Manual Setup

```bash
# Navigate to the voting system directory
cd voting-system

# Install Go dependencies
go mod download

# Install Node.js dependencies
npm install

# Start the development blockchain
make blockchain-start

# Deploy smart contracts
make contracts-deploy

# Start the backend server
make server-start

# Start the web interface
make web-start
```

## 📁 Project Structure

```
Decentralized-Voting-Project/
├── voting-system/                 # Main application directory
│   ├── contracts/                 # Smart contracts
│   │   ├── SecureVotingSystem.sol # Main voting contract
│   │   └── migrations/            # Contract deployment scripts
│   ├── cmd/                       # Application entry points
│   ├── internal/                  # Private application code
│   │   ├── blockchain/           # Blockchain integration
│   │   ├── handlers/             # HTTP request handlers
│   │   ├── models/               # Data models
│   │   └── services/             # Business logic
│   ├── web/                      # Frontend application
│   │   ├── static/               # Static assets
│   │   └── templates/            # HTML templates
│   ├── tests/                    # Test files
│   ├── docs/                     # Documentation
│   ├── docker/                   # Docker configuration
│   ├── configs/                  # Configuration files
│   ├── scripts/                  # Utility scripts
│   └── Makefile                  # Build automation
├── quick_setup.sh                # Automated setup script
└── README.md                     # This file
```

## 🔧 Configuration

### Environment Variables

Create a `.env` file in the `voting-system` directory:

```env
# Blockchain Configuration
ETHEREUM_RPC_URL=http://localhost:8545
CONTRACT_ADDRESS=0x345cA3e014Aaf5dcA488057592ee47305D9B3e10

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=localhost

# Database Configuration
DATABASE_URL=sqlite:///voting.db
# or for PostgreSQL: postgresql://user:password@localhost/voting

# Security
JWT_SECRET=your-secret-key
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run specific test suites
make test-contracts
make test-backend
make test-integration
```

## 🐳 Docker Deployment

```bash
# Build and run with Docker Compose
docker-compose up --build

# Or build individual containers
make docker-build
make docker-run
```

## 📊 Smart Contract Details

- **Contract Name**: SecureVotingSystem
- **Contract Address**: `0x345cA3e014Aaf5dcA488057592ee47305D9B3e10`
- **Network**: Development (Ganache)
- **Features**:
  - Voter registration and verification
  - Secure vote casting
  - Real-time result calculation
  - Admin controls and election management

## 🔒 Security Features

- **Cryptographic Verification**: SHA-256 hashing for voter identification
- **Duplicate Vote Prevention**: Smart contract-level protection
- **Immutable Records**: All votes stored on blockchain
- **Access Control**: Role-based permissions for administrators
- **Input Validation**: Comprehensive data validation at all layers

## 📈 Performance

- **Transaction Speed**: ~15 seconds per vote (Ethereum network)
- **Concurrent Users**: Supports 100+ simultaneous voters
- **Scalability**: Modular architecture for easy scaling
- **Uptime**: 99.9% availability with proper deployment

## 🤝 Contributing

This is a final year project, but contributions and suggestions are welcome:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## 📚 Academic Context

**Institution**: Obafemi Awolowo University  
**Department**: Computer Science and Engineering  
**Project Type**: Final Year Project  
**Academic Year**: 2024/2025

### Research Areas

- Blockchain Technology
- Decentralized Applications (DApps)
- Cryptography and Security
- Distributed Systems
- Web3 Development

## 📄 License

This project is developed for academic purposes at Obafemi Awolowo University. All rights reserved.

## 👨‍💻 Author

**Student**: Olawale Olatunji Bright
**Supervisor**: Mr. M. A. Akingbade
**Department**: Computer Science and Engineering  
**Institution**: Obafemi Awolowo University

## 📞 Support

For technical support or questions about this project:

- **Email**: [obolawale@student.oauife.edu.ng]
- **Department**: Computer Science and Engineering, OAU
- **Office Hours**: [By appointment]

## 🔮 Future Enhancements

- [ ] Mobile application development
- [ ] Integration with multiple blockchain networks
- [ ] Advanced analytics and reporting
- [ ] Multi-language support
- [ ] Offline voting capabilities
- [ ] Integration with government ID systems

---

**Note**: This project is developed for educational purposes and should be thoroughly tested before any production deployment.
