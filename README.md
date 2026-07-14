# Blockchain Simulator

A simple, educational blockchain simulator written in Go. This project is designed to demonstrate the fundamental mechanics of a blockchain, including transactions, mining blocks via proof-of-work, cryptographic hashing, and validating the integrity of the chain through an interactive Command Line Interface (CLI).

## Prerequisites

- Go (1.20 or newer)

## Running the Project

You can start the interactive CLI directly using the Go toolchain.

From the root of the project, run:
```bash
go run ./cmd/blockchainsimulator
```

## Available Commands

Once the CLI is running, you can interact with your local blockchain using the following commands:

- `createwallet <name>`        - Create a new wallet and save it to disk.
- `loadwallet <name>`          - Load an existing wallet from disk.
- `wallets`                    - List all saved wallets.
- `mywallet`                   - View your current wallet address and balance.
- `faucet <amount>`            - Request free funds from the Faucet.
- `addtx <to> <amount>`        - Send funds to an address (uses current wallet).
- `mine`                       - Mine pending transactions into a new block.
- `pool`                       - View all pending transactions.
- `balances`                   - View all account balances.
- `validate`                   - Validate the cryptographic integrity of the entire blockchain.
- `print`                      - Visualize the blockchain structure and block details.
- `help`                       - Display all available commands.
- `clear`                      - Clear the terminal screen.
- `exit`                       - Exit the CLI safely and save the chain state.

## Simulating a Transaction

When you first start the simulator, your chain will be empty, and all accounts will have a zero balance. You can get initial funds using the special `faucet` command, which dispenses funds to your active wallet.

1. **Create your first wallet:**
   ```text
   createwallet Alice
   ```

2. **Mint initial funds from the FAUCET:**
   ```text
   faucet 100
   ```

3. **Check the pending transaction pool:**
   ```text
   pool
   ```

4. **Mine the transaction into a new block:**
   ```text
   mine
   ```

5. **Create a second wallet to transfer funds:**
   ```text
   createwallet Bob
   ```

6. **Load the first wallet and transfer money:**
   ```text
   loadwallet Alice
   addtx <Bob's Address> 50
   pool
   mine
   balances
   ```

## Architecture & Design Decisions

- **Account-Based Ledger**: Instead of the UTXO (Unspent Transaction Output) model used by Bitcoin, this simulator employs an account-based model. Balances are dynamically calculated by iterating through historical blocks and deducting pending transactions to proactively prevent double-spending.
- **Merkle Root for Transactions**: Transactions are hashed independently and paired up into a Merkle tree. Only the fixed-length Merkle root is included in the block header. This avoids hashing the raw transaction list directly and secures the transaction set.
- **Double SHA-256 Hashing**: The block hash is calculated using double SHA-256 (`SHA256(SHA256(data))`). This matches Bitcoin's approach and provides protection against length-extension attacks.
- **Digital Signatures (ECDSA)**: Transactions are cryptographically signed using the Elliptic Curve Digital Signature Algorithm (secp256r1). This ensures that only the true owner of a wallet's private key can authorize funds from that address, preventing spoofing.
- **Wallet Keystore**: Wallets are generated with public/private key pairs and stored securely on disk in a JSON-based keystore, allowing users to persist their identities.
- **Block Header Structure**: The final block hash incorporates the Previous Hash, Merkle Root, Timestamp, Difficulty, and Nonce, effectively tying the entire chain's history into each block.
- **Proof-of-Work Consensus**: A basic proof-of-work system is implemented. Miners increment a `uint32` Nonce to find a hash that meets a specific difficulty target (leading zeros).
- **ExtraNonce for Infinite Mining Space**: If the standard `uint32` Nonce space overflows during mining, the system increments an `ExtraNonce` inside the Coinbase transaction and recalculates the Merkle Root, creating infinite search space for a valid hash.
- **Coinbase Protection**: The system reserves the `COINBASE` sender string strictly for block mining rewards and rejects any user-submitted transactions attempting to spoof it.
- **Data Persistence**: The blockchain state is automatically saved to a JSON file (`storage/chain.json`) when exiting the CLI safely, and loaded when starting. This allows you to persist your blockchain state across multiple sessions in a human-readable format.
- **Static Genesis Anchor**: The first block of the chain (Genesis block) is purposely hardcoded with a fixed timestamp and properties. Just like in real blockchains (e.g., Bitcoin), this provides a consistent, universally recognizable anchor for the entire chain.

## Known Limitations

- **No Peer Consensus or Finality**: The simulator runs as a single local process without a network of peers. There is no real trust model or concept of finality, as there's only one copy of the truth and no longest-chain rule applied across a decentralized network.
- **ExtraNonce Implementation**: `ExtraNonce` is currently implemented as its own distinct typed field. In production blockchains like Bitcoin, the extra nonce is often stored within the empty signature script space of the Coinbase transaction.
- **Static Mining Parameters**: The mining reward is statically hardcoded (50 coins to a "Miner" address). There are no transaction fees, block reward halvings, or dynamic difficulty adjustments based on network hash rate or block time.
