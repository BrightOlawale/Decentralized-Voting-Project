#!/bin/bash

# Generate Go bindings from Solidity smart contracts
# This script compiles the contracts and generates Go bindings

set -e

echo "🔧 Generating Go bindings from smart contracts..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if required tools are installed
check_tool() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}Error: $1 is not installed${NC}"
        echo "Please install $1 and try again"
        exit 1
    fi
}

echo "📋 Checking required tools..."
check_tool "truffle"
check_tool "abigen"

# Compile contracts with Truffle
echo -e "${YELLOW}📦 Compiling smart contracts...${NC}"
truffle compile

# Check if compilation was successful
if [ ! -d "build/contracts" ]; then
    echo -e "${RED}Error: Contract compilation failed${NC}"
    exit 1
fi

# Create bindings directory if it doesn't exist
mkdir -p internal/blockchain

# Generate Go bindings
echo -e "${YELLOW}🔗 Generating Go bindings...${NC}"

# Generate bindings for SecureVotingSystem contract
if [ -f "build/contracts/SecureVotingSystem.json" ]; then
    echo "Generating bindings for SecureVotingSystem..."
    
    # Extract ABI and Bytecode from the JSON file
    cat build/contracts/SecureVotingSystem.json | jq -r .abi > temp_abi.json
    cat build/contracts/SecureVotingSystem.json | jq -r .bytecode > temp_bytecode.txt
    
    # Generate Go bindings
    abigen \
        --abi temp_abi.json \
        --bin temp_bytecode.txt \
        --pkg blockchain \
        --type SecureVotingSystem \
        --out internal/blockchain/contracts.go
    
    # Clean up temporary files
    rm temp_abi.json temp_bytecode.txt
    
    echo -e "${GREEN}✅ SecureVotingSystem bindings generated successfully${NC}"
else
    echo -e "${RED}Error: SecureVotingSystem.json not found${NC}"
    exit 1
fi

# Generate bindings for OpenZeppelin contracts if needed
if [ -f "build/contracts/Ownable.json" ]; then
    echo "Generating bindings for Ownable..."
    
    cat build/contracts/Ownable.json | jq -r .abi > temp_ownable_abi.json
    
    abigen \
        --abi temp_ownable_abi.json \
        --pkg blockchain \
        --type Ownable \
        --out internal/blockchain/ownable.go
    
    rm temp_ownable_abi.json
    
    echo -e "${GREEN}✅ Ownable bindings generated successfully${NC}"
fi

# Update contract address in configuration
if [ -f "build/contracts/SecureVotingSystem.json" ]; then
    # Extract the contract address from the networks section
    CONTRACT_ADDRESS=$(cat build/contracts/SecureVotingSystem.json | jq -r '.networks | to_entries | .[0].value.address // empty')
    
    if [ ! -z "$CONTRACT_ADDRESS" ] && [ "$CONTRACT_ADDRESS" != "null" ]; then
        echo -e "${YELLOW}📝 Updating contract address in configuration...${NC}"
        
        # Update .env file
        if [ -f ".env" ]; then
            # Remove existing CONTRACT_ADDRESS line and add new one
            grep -v "^CONTRACT_ADDRESS=" .env > .env.tmp || true
            echo "CONTRACT_ADDRESS=$CONTRACT_ADDRESS" >> .env.tmp
            mv .env.tmp .env
            echo -e "${GREEN}✅ Updated CONTRACT_ADDRESS in .env: $CONTRACT_ADDRESS${NC}"
        fi
        
        # Update YAML configs
        for config_file in configs/terminal.yaml configs/server.yaml; do
            if [ -f "$config_file" ]; then
                # Use sed to replace the contract_address line
                sed -i.bak "s/contract_address: .*/contract_address: \"$CONTRACT_ADDRESS\"/" "$config_file"
                rm "$config_file.bak" 2>/dev/null || true
                echo -e "${GREEN}✅ Updated contract address in $config_file${NC}"
            fi
        done
    else
        echo -e "${YELLOW}⚠️  Contract address not found in build artifacts${NC}"
        echo "You may need to deploy the contract first with: truffle migrate"
    fi
fi

# Verify generated files
echo -e "${YELLOW}🔍 Verifying generated files...${NC}"

if [ -f "internal/blockchain/contracts.go" ]; then
    # Check if the file contains expected content
    if grep -q "SecureVotingSystem" internal/blockchain/contracts.go; then
        echo -e "${GREEN}✅ Generated bindings appear to be valid${NC}"
        
        # Show some stats about the generated file
        LINES=$(wc -l < internal/blockchain/contracts.go)
        echo "Generated file contains $LINES lines"
        
        # Show the main struct definition
        echo -e "${YELLOW}📋 Generated contract interface:${NC}"
        grep -A 5 "type SecureVotingSystem struct" internal/blockchain/contracts.go || true
        
    else
        echo -e "${RED}❌ Generated bindings appear to be invalid${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ Bindings file was not generated${NC}"
    exit 1
fi

# Generate a summary file
cat > CONTRACT_INFO.md << EOF
# Smart Contract Deployment Information

## Contract Details
- **Contract Name**: SecureVotingSystem
- **Contract Address**: \`$CONTRACT_ADDRESS\`
- **Network**: Development (Ganache)
- **Generated**: $(date)

## Files Generated
- \`internal/blockchain/contracts.go\` - Main contract bindings
- \`internal/blockchain/ownable.go\` - Ownable contract bindings (if applicable)

## Usage in Go Code
\`\`\`go
import "voting-system/internal/blockchain"

// Create contract instance
client, _ := ethclient.Dial("http://localhost:8545")
contract, _ := blockchain.NewSecureVotingSystem(
    common.HexToAddress("$CONTRACT_ADDRESS"), 
    client
)

// Use contract methods
hasVoted, _ := contract.HasVoterVoted(nil, voterHash)
\`\`\`

## Next Steps
1. Update your Go code to use the generated bindings
2. Test the contract integration
3. Deploy to testnet when ready
EOF

echo -e "${GREEN}📄 Created CONTRACT_INFO.md with deployment details${NC}"

echo ""
echo -e "${GREEN}🎉 Contract bindings generation completed successfully!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Update your Go code to import the generated bindings"
echo "2. Test the contract integration with: go test ./internal/blockchain"
echo "3. Run the application with: make run-terminal"
echo ""
echo -e "${YELLOW}Contract Address:${NC} $CONTRACT_ADDRESS"
echo -e "${YELLOW}Bindings File:${NC} internal/blockchain/contracts.go"