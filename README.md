# Blockchain Simulator

A simple blockchain simulator written in Go. This simulator allows you to understand the fundamental concepts of blockchain such as transactions, mining blocks, and validating the chain via an interactive Command Line Interface (CLI).

## Prerequisites

- Go (1.20+)

## Running the Project

To start the interactive Blockchain Simulator CLI, run the following command from the root of the project:

```bash
go run cmd/blockchainsimulator/main.go
```

## Available Commands

Once the CLI is running, you can interact with the blockchain using the following commands:

- `addtx <from> <to> <amount>` - Add a new transaction to the pending pool
- `mine`                       - Mine pending transactions into a new block
- `pool`                       - View all pending transactions
- `balances`                   - View account balances
- `validate`                   - Validate the integrity of the blockchain
- `print`                      - Visualize the blockchain structure
- `help`                       - Display available commands
- `clear`                      - Clear the terminal screen
- `exit`                       - Exit the Blockchain CLI
