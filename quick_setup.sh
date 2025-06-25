#!/bin/bash

# Universal Golang Voting System Setup Script
# Supports: Ubuntu/Debian, CentOS/RHEL/Fedora, Arch Linux, macOS
# This script sets up the entire project from scratch

set -e  # Exit on any error

echo "🚀 Starting Golang Voting System Setup (Universal)..."
echo "====================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect operating system and package manager
detect_os() {
    print_status "Detecting operating system..."
    
    OS=""
    PACKAGE_MANAGER=""
    
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="linux"
        if command -v apt-get &> /dev/null; then
            PACKAGE_MANAGER="apt"
            DISTRO="debian"
        elif command -v dnf &> /dev/null; then
            PACKAGE_MANAGER="dnf"
            DISTRO="fedora"
        elif command -v yum &> /dev/null; then
            PACKAGE_MANAGER="yum"
            DISTRO="rhel"
        elif command -v pacman &> /dev/null; then
            PACKAGE_MANAGER="pacman"
            DISTRO="arch"
        elif command -v zypper &> /dev/null; then
            PACKAGE_MANAGER="zypper"
            DISTRO="opensuse"
        else
            print_error "Unsupported Linux distribution"
            exit 1
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
        PACKAGE_MANAGER="brew"
        DISTRO="macos"
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        OS="windows"
        PACKAGE_MANAGER="choco"
        DISTRO="windows"
    else
        print_error "Unsupported operating system: $OSTYPE"
        exit 1
    fi
    
    print_success "Detected: $OS ($DISTRO) with $PACKAGE_MANAGER package manager"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Installing Go..."
        install_go
    else
        GO_VERSION=$(go version | cut -d ' ' -f 3 | cut -d 'o' -f 2)
        print_success "Go $GO_VERSION is installed"
    fi
}

# Install Go if not present
install_go() {
    print_status "Installing Go..."
    
    case $OS in
        "linux")
            GO_VERSION="1.21.5"
            case $(uname -m) in
                x86_64) ARCH="amd64" ;;
                aarch64) ARCH="arm64" ;;
                armv6l) ARCH="armv6l" ;;
                *) print_error "Unsupported architecture"; exit 1 ;;
            esac
            
            wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz"
            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-${ARCH}.tar.gz"
            rm "go${GO_VERSION}.linux-${ARCH}.tar.gz"
            
            # Add to PATH
            echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
            export PATH=$PATH:/usr/local/go/bin
            ;;
        "macos")
            if command -v brew &> /dev/null; then
                brew install go
            else
                print_error "Homebrew not found. Please install Homebrew first or install Go manually."
                exit 1
            fi
            ;;
        "windows")
            print_error "Please install Go manually from https://golang.org/dl/"
            exit 1
            ;;
    esac
    
    print_success "Go installed successfully"
}

# Check if Node.js is installed
check_node() {
    if ! command -v node &> /dev/null; then
        print_warning "Node.js is not installed. Installing..."
        install_node
    else
        NODE_VERSION=$(node --version)
        print_success "Node.js $NODE_VERSION is installed"
    fi
}

# Install Node.js if not present
install_node() {
    print_status "Installing Node.js..."
    
    case $PACKAGE_MANAGER in
        "apt")
            curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
            sudo apt-get install -y nodejs
            ;;
        "dnf"|"yum")
            curl -fsSL https://rpm.nodesource.com/setup_18.x | sudo bash -
            sudo $PACKAGE_MANAGER install -y nodejs npm
            ;;
        "pacman")
            sudo pacman -S --noconfirm nodejs npm
            ;;
        "zypper")
            sudo zypper install -y nodejs18 npm18
            ;;
        "brew")
            brew install node
            ;;
        "choco")
            choco install nodejs
            ;;
    esac
    
    print_success "Node.js installed successfully"
}

# Install system dependencies
install_dependencies() {
    print_status "Installing system dependencies..."
    
    case $PACKAGE_MANAGER in
        "apt")
            sudo apt-get update -qq
            sudo apt-get install -y \
                git \
                make \
                build-essential \
                sqlite3 \
                postgresql \
                postgresql-contrib \
                curl \
                wget \
                unzip
            ;;
        "dnf")
            sudo dnf groupinstall -y "Development Tools"
            sudo dnf install -y \
                git \
                make \
                sqlite \
                postgresql \
                postgresql-server \
                curl \
                wget \
                unzip
            ;;
        "yum")
            sudo yum groupinstall -y "Development Tools"
            sudo yum install -y \
                git \
                make \
                sqlite \
                postgresql \
                postgresql-server \
                curl \
                wget \
                unzip
            ;;
        "pacman")
            sudo pacman -S --noconfirm \
                git \
                make \
                base-devel \
                sqlite \
                postgresql \
                curl \
                wget \
                unzip
            ;;
        "zypper")
            sudo zypper install -y \
                git \
                make \
                gcc \
                sqlite3 \
                postgresql \
                postgresql-server \
                curl \
                wget \
                unzip
            ;;
        "brew")
            brew install \
                git \
                make \
                sqlite \
                postgresql \
                curl \
                wget \
                unzip
            ;;
        "choco")
            choco install git make sqlite postgresql curl wget
            ;;
        *)
            print_warning "Package manager not supported. Please install manually:"
            print_warning "Required: git, make, sqlite, postgresql, curl, wget"
            ;;
    esac
    
    print_success "System dependencies installed"
}

# Setup PostgreSQL database
setup_postgresql() {
    print_status "Setting up PostgreSQL database..."
    
    case $DISTRO in
        "debian")
            sudo systemctl start postgresql
            sudo systemctl enable postgresql
            ;;
        "fedora"|"rhel")
            if [ "$PACKAGE_MANAGER" = "dnf" ] || [ "$PACKAGE_MANAGER" = "yum" ]; then
                # Initialize database for RHEL/CentOS
                if [ ! -f /var/lib/pgsql/data/postgresql.conf ]; then
                    sudo postgresql-setup initdb
                fi
                sudo systemctl start postgresql
                sudo systemctl enable postgresql
            fi
            ;;
        "arch")
            sudo systemctl start postgresql
            sudo systemctl enable postgresql
            # Initialize database if not exists
            if [ ! -d /var/lib/postgres/data ]; then
                sudo -u postgres initdb -D /var/lib/postgres/data
            fi
            ;;
        "opensuse")
            sudo systemctl start postgresql
            sudo systemctl enable postgresql
            ;;
        "macos")
            if command -v brew &> /dev/null; then
                brew services start postgresql
                # Create database cluster if it doesn't exist
                if [ ! -d /opt/homebrew/var/postgres ]; then
                    initdb /opt/homebrew/var/postgres
                fi
            fi
            ;;
        "windows")
            print_warning "Please start PostgreSQL service manually on Windows"
            ;;
    esac
    
    # Create user and database
    create_postgres_user
    
    print_success "PostgreSQL configured"
}

# Create PostgreSQL user and database
create_postgres_user() {
    print_status "Creating PostgreSQL user and database..."
    
    # Try different methods to create user based on OS
    if [[ "$OS" == "macos" ]]; then
        # macOS with Homebrew
        createuser -s voting_user 2>/dev/null || true
        createdb voting_system 2>/dev/null || true
        psql voting_system -c "ALTER USER voting_user WITH PASSWORD 'voting_password';" 2>/dev/null || true
    else
        # Linux systems
        sudo -u postgres psql << 'EOF' 2>/dev/null || print_warning "Database setup may need manual configuration"
CREATE USER voting_user WITH PASSWORD 'voting_password';
CREATE DATABASE voting_system OWNER voting_user;
GRANT ALL PRIVILEGES ON DATABASE voting_system TO voting_user;
\q
EOF
    fi
}

# Create project structure
create_project_structure() {
    print_status "Creating project directory structure..."
    
    PROJECT_NAME="voting-system"
    
    if [ -d "$PROJECT_NAME" ]; then
        print_warning "Directory $PROJECT_NAME already exists. Removing..."
        rm -rf "$PROJECT_NAME"
    fi
    
    mkdir -p "$PROJECT_NAME"
    cd "$PROJECT_NAME"
    
    # Create directory structure
    mkdir -p {cmd/{terminal,server,admin},contracts/{migrations,bindings},internal/{biometric,blockchain,database,encryption,hardware,api},pkg/{config,logger,utils},web/{static/{css,js,images},templates},scripts,configs,tests/{integration,unit},docker,docs,logs,bin}
    
    print_success "Project structure created in $(pwd)"
}

# Initialize Go module and dependencies
setup_go_project() {
    print_status "Initializing Go module..."
    
    go mod init voting-system
    
    print_status "Installing Go dependencies..."
    
    # Core dependencies
    go get github.com/ethereum/go-ethereum@latest
    go get github.com/gin-gonic/gin@latest
    go get github.com/gorilla/websocket@latest
    go get gorm.io/gorm@latest
    go get gorm.io/driver/sqlite@latest
    go get gorm.io/driver/postgres@latest
    go get github.com/joho/godotenv@latest
    go get github.com/golang-jwt/jwt/v4@latest
    go get github.com/redis/go-redis/v9@latest
    go get github.com/sirupsen/logrus@latest
    go get github.com/spf13/viper@latest
    go get github.com/stretchr/testify@latest
    go get golang.org/x/crypto@latest
    go get gopkg.in/yaml.v3@latest
    
    print_success "Go dependencies installed"
}

# Setup blockchain tools
setup_blockchain_tools() {
    print_status "Installing blockchain development tools..."
    
    # Install Truffle and Ganache
    npm install -g truffle ganache-cli
    
    # Initialize npm project
    npm init -y
    npm install @openzeppelin/contracts@4.9.0 truffle
    
    print_success "Blockchain tools installed"
}

# Create configuration files
create_config_files() {
    print_status "Creating configuration files..."
    
    # Terminal configuration
    cat > configs/terminal.yaml << 'EOF'
server:
  port: 8080
  host: "0.0.0.0"
  
database:
  type: "sqlite"
  path: "./terminal.db"
  
blockchain:
  network_url: "http://localhost:8545"
  contract_address: ""
  private_key: ""
  
biometric:
  fingerprint_device: "/dev/ttyUSB0"
  quality_threshold: 0.8
  match_threshold: 0.85
  
hardware:
  display_device: "/dev/fb0"
  printer_device: "/dev/ttyUSB1"
  
encryption:
  key: "your-32-byte-encryption-key-here!!"
  
logging:
  level: "info"
  file: "./logs/terminal.log"
EOF

    # Server configuration
    cat > configs/server.yaml << 'EOF'
server:
  port: 8081
  host: "0.0.0.0"
  
database:
  type: "postgres"
  host: "localhost"
  port: 5432
  user: "voting_user"
  password: "voting_password"
  dbname: "voting_system"
  
blockchain:
  network_url: "http://localhost:8545"
  contract_address: ""
  private_key: ""
  
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  
logging:
  level: "info"
  file: "./logs/server.log"
EOF

    # Environment file
    cat > .env << 'EOF'
# Blockchain Configuration
PRIVATE_KEY=0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318
CONTRACT_ADDRESS=""
NETWORK_URL=http://localhost:8545

# Encryption
ENCRYPTION_KEY=your-32-byte-encryption-key-here!!

# Database
DB_PASSWORD=voting_password
DB_USER=voting_user

# JWT Secret
JWT_SECRET=your-jwt-secret-key-here

# Admin Credentials
ADMIN_USERNAME=admin
ADMIN_PASSWORD=secure_admin_password
EOF

    print_success "Configuration files created"
}

# Create Makefile with OS-specific commands
create_makefile() {
    print_status "Creating Makefile..."
    
    cat > Makefile << 'EOF'
.PHONY: build test clean run-terminal run-server deploy-contracts

# Build applications
build:
	go build -o bin/terminal cmd/terminal/main.go
	go build -o bin/server cmd/server/main.go
	go build -o bin/admin cmd/admin/main.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf contracts/bindings/

# Run terminal application
run-terminal:
	go run cmd/terminal/main.go

# Run server application
run-server:
	go run cmd/server/main.go

# Deploy smart contracts
deploy-contracts:
	truffle compile
	truffle migrate --reset

# Generate Go bindings from smart contracts
generate-bindings:
	abigen --sol contracts/VotingSystem.sol --pkg contracts --out internal/blockchain/contracts.go

# Setup development environment
setup-dev:
	go mod tidy
	mkdir -p logs bin
	chmod +x scripts/*.sh

# Start Ganache blockchain
start-blockchain:
	ganache-cli --deterministic --accounts 10 --host 0.0.0.0 --port 8545 --mnemonic "candy maple cake sugar pudding cream honey rich smooth crumble sweet treat"

# Initialize database
init-db:
	go run scripts/migrate.go

# Full setup
setup: setup-dev init-db
	@echo "Setup completed!"

# Development workflow
dev: start-blockchain deploy-contracts run-terminal
EOF

    print_success "Makefile created"
}

# Create basic Go files
create_go_files() {
    print_status "Creating basic Go source files..."
    
    # Create basic config package
    cat > pkg/config/config.go << 'EOF'
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Server struct {
        Port string `mapstructure:"port"`
        Host string `mapstructure:"host"`
    } `mapstructure:"server"`
    
    Database struct {
        Type     string `mapstructure:"type"`
        Path     string `mapstructure:"path"`
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        User     string `mapstructure:"user"`
        Password string `mapstructure:"password"`
        DBName   string `mapstructure:"dbname"`
    } `mapstructure:"database"`
    
    Blockchain struct {
        NetworkURL      string `mapstructure:"network_url"`
        ContractAddress string `mapstructure:"contract_address"`
        PrivateKey      string `mapstructure:"private_key"`
    } `mapstructure:"blockchain"`
    
    Biometric struct {
        FingerprintDevice string  `mapstructure:"fingerprint_device"`
        QualityThreshold  float64 `mapstructure:"quality_threshold"`
        MatchThreshold    float64 `mapstructure:"match_threshold"`
    } `mapstructure:"biometric"`
}

func LoadConfig(path string) (*Config, error) {
    viper.SetConfigFile(path)
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
EOF

    # Create basic main files
    cat > cmd/terminal/main.go << 'EOF'
package main

import (
    "fmt"
    "log"
    "net/http"
    
    "voting-system/pkg/config"
    "github.com/gin-gonic/gin"
)

func main() {
    cfg, err := config.LoadConfig("configs/terminal.yaml")
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    r := gin.Default()
    r.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "message": "Voting Terminal is running",
            "status":  "active",
        })
    })
    
    addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
    fmt.Printf("Starting terminal server on %s\n", addr)
    log.Fatal(http.ListenAndServe(addr, r))
}
EOF

    cat > cmd/server/main.go << 'EOF'
package main

import (
    "fmt"
    "log"
    "net/http"
    
    "voting-system/pkg/config"
    "github.com/gin-gonic/gin"
)

func main() {
    cfg, err := config.LoadConfig("configs/server.yaml")
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    r := gin.Default()
    r.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "message": "Central Voting Server is running",
            "status":  "active",
        })
    })
    
    addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
    fmt.Printf("Starting central server on %s\n", addr)
    log.Fatal(http.ListenAndServe(addr, r))
}
EOF

    print_success "Basic Go files created"
}

# Create smart contract and Truffle configuration
create_smart_contract() {
    print_status "Creating smart contract configuration..."
    
    # Create truffle config
    cat > truffle-config.js << 'EOF'
module.exports = {
  networks: {
    development: {
      host: "127.0.0.1",
      port: 8545,
      network_id: "*",
      gas: 6721975,
      gasPrice: 20000000000,
    },
  },
  compilers: {
    solc: {
      version: "0.8.19",
      settings: {
        optimizer: {
          enabled: true,
          runs: 200
        },
      }
    }
  },
  contracts_directory: './contracts',
  contracts_build_directory: './contracts/bindings'
};
EOF

    # Create migration files
    cat > contracts/migrations/1_initial_migration.js << 'EOF'
const Migrations = artifacts.require("Migrations");

module.exports = function (deployer) {
  deployer.deploy(Migrations);
};
EOF

    print_success "Smart contract configuration created"
}

# Create README file
create_readme() {
    print_status "Creating README file..."
    
    cat > README.md << 'EOF'
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
EOF

    print_success "README created"
}

# Final setup and validation
final_setup() {
    print_status "Running final setup and validation..."
    
    # Tidy Go modules
    go mod tidy
    
    # Set permissions for scripts
    find scripts -name "*.sh" -exec chmod +x {} \; 2>/dev/null || true
    
    # Create initial SQLite database
    touch terminal.db
    
    # Validate Go installation
    if go version &>/dev/null; then
        print_success "Go installation validated"
    else
        print_error "Go installation failed"
        exit 1
    fi
    
    # Validate Node.js installation
    if node --version &>/dev/null; then
        print_success "Node.js installation validated"
    else
        print_error "Node.js installation failed"
        exit 1
    fi
    
    print_success "Final setup completed"
}

# Main execution function
main() {
    echo "🔍 Detecting system and checking prerequisites..."
    detect_os
    check_go
    check_node
    
    echo ""
    echo "📦 Installing system dependencies..."
    install_dependencies
    
    echo ""
    echo "🗄️  Setting up database..."
    setup_postgresql
    
    echo ""
    echo "🏗️  Creating project structure..."
    create_project_structure
    setup_go_project
    setup_blockchain_tools
    
    echo ""
    echo "⚙️  Configuring project..."
    create_config_files
    create_makefile
    create_go_files
    create_smart_contract
    create_readme
    
    echo ""
    echo "🔧 Running final setup..."
    final_setup
    
    echo ""
    echo "============================================"
    echo "🎉 PROJECT SETUP COMPLETED SUCCESSFULLY! 🎉"
    echo "============================================"
    echo ""
    echo "📁 Project location: $(pwd)"
    echo "💻 Operating System: $OS ($DISTRO)"
    echo "📦 Package Manager: $PACKAGE_MANAGER"
    echo ""
    echo "🚀 Quick start commands:"
    echo "  1. Start blockchain:    make start-blockchain"
    echo "  2. Deploy contracts:    make deploy-contracts"
    echo "  3. Run terminal:        make run-terminal"
    echo "  4. Run server:          make run-server"
    echo ""
    echo "📚 Check the README.md for detailed instructions"
    echo "🔧 Configuration files are in the configs/ directory"
    echo ""
    case $OS in
        "linux")
            echo "🐧 Linux-specific notes:"
            echo "  - PostgreSQL service should be running"
            echo "  - Check firewall settings if needed"
            ;;
        "macos")
            echo "🍎 macOS-specific notes:"
            echo "  - PostgreSQL installed via Homebrew"
            echo "  - May need to accept security prompts"
            ;;
        "windows")
            echo "🪟 Windows-specific notes:"
            echo "  - Some features may require manual configuration"
            echo "  - Use Git Bash or WSL for best experience"
            ;;
    esac
    echo ""
    echo "Happy coding! 🚀"
}

# Check if script is being run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
