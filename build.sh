#!/bin/bash

# Define colors for pretty output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print a banner
print_banner() {
    echo -e "${GREEN}=========================================${NC}"
    echo -e "${YELLOW}          Memorandum Build Script        ${NC}"
    echo -e "${GREEN}=========================================${NC}"
}

# Function to build the main project
build_main() {
    echo -e "${YELLOW}Building the main project...${NC}"
    go build -o Memorandum ./
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Main project built successfully as 'Memorandum'!${NC}"
    else
        echo -e "${RED}Failed to build the main project.${NC}"
        exit 1
    fi
}

# Function to build the CLI
build_cli() {
    echo -e "${YELLOW}Building the CLI...${NC}"
    go build -o Memorandum-cli ./cli
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}CLI built successfully as 'Memorandum-cli'!${NC}"
    else
        echo -e "${RED}Failed to build the CLI.${NC}"
        exit 1
    fi
}

# Main script execution
print_banner
build_main
build_cli

echo -e "${GREEN}All builds completed successfully!${NC}"
