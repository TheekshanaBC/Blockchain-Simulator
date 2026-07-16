# Blockchain Simulator

An educational blockchain simulator written in Go, demonstrating the core mechanics of a production-quality blockchain: cryptographic hashing, ECDSA digital signatures, Merkle trees, Proof-of-Work mining, dynamic difficulty retargeting, replay-attack protection, and a full-featured interactive CLI.

## Prerequisites

- **Go 1.22** or newer

## Project Structure

```
blockchain-simulator/
├── cmd/
│   ├── blockchainsimulator/    # Main CLI entry point
│   └── mining_experiment/      # Standalone PoW benchmarking tool
├── internal/
│   ├── block/                  # Block, Transaction, Merkle, Mining, Signing
│   ├── chain/                  # Chain state, validation, difficulty, faucet
│   ├── cli/                    # Interactive CLI (commands, display)
│   ├── ledger/                 # Account-based balance & sequence tracking
│   ├── storage/                # JSON persistence (atomic writes)
│   └── wallet/                 # ECDSA key generation & keystore
└── data/                       # Runtime data (chain.json, wallet.json)
```

## Running the Simulator

Start the interactive CLI from the project root:

```bash
go run ./cmd/blockchainsimulator
```

### Command-Line Flags

The simulator accepts optional flags to tune chain parameters at startup. These are only applied when **creating a new chain**; a loaded chain retains its own saved parameters.

| Flag | Default | Description |
|---|---|---|
| `-diff` | `4` | Initial mining difficulty (leading zeros required in hash) |
| `-retarget-window` | `4` | Number of blocks between difficulty retargets |
| `-target-block-time` | `10` | Target time per block in seconds |
| `-min-diff` | `3` | Minimum allowed difficulty |
| `-max-diff` | `8` | Maximum allowed difficulty |

**Example — fast/easy chain for experimentation:**

```bash
go run ./cmd/blockchainsimulator -diff 2 -min-diff 1 -max-diff 5 -retarget-window 3 -target-block-time 5
```

## Available CLI Commands

Once the CLI is running, interact with your blockchain using these commands:

| Command | Description |
|---|---|
| `createwallet <name>` | Generate a new ECDSA wallet and save it to disk |
| `loadwallet <name>` | Load an existing wallet from the keystore |
| `wallets` | List all saved wallets and their addresses |
| `mywallet` | Show your active wallet's address and confirmed balance |
| `faucet <amount>` | Request free test funds (max 1,000 per request; 5,000 lifetime per address) |
| `addtx <to> <amount>` | Sign and submit a transaction to the pending pool |
| `mine` | Mine all pending transactions into a new block |
| `pool` | View all pending transactions |
| `balances` | View all confirmed account balances |
| `validate` | Cryptographically validate the entire blockchain |
| `print` | Visualize the chain structure with block details |
| `help` | Display all available commands |
| `clear` | Clear the terminal screen |
| `exit` | Save the chain to disk and exit |

## Simulating a Transaction (Walkthrough)

All accounts start at zero. Use the `faucet` to bootstrap initial funds.

**1. Create your first wallet:**
```
createwallet Alice
```

**2. Request test funds from the faucet:**
```
faucet 100
```

**3. Mine the faucet transaction into a block:**
```
mine
```

**4. Create a second wallet:**
```
createwallet Bob
```

**5. Send funds to Bob:**
```
loadwallet Alice
addtx <Bob's Address> 50
mine
balances
```

**6. Validate the chain:**
```
validate
```

## Mining Experiment Tool

A standalone benchmarking utility measures mining time and hash attempts across difficulty levels 1–8 using the parallel PoW implementation.

```bash
go run ./cmd/mining_experiment
```

Sample output:

```
Difficulty   | Time Taken   | Hashes Tried
----------------------------------------------
1            | 0s           | 1
2            | 0s           | 43
3            | 1ms          | 412
4            | 12ms         | 3218
...
```

## Architecture & Design Decisions

### Block & Transaction Structure

- **`Transaction`**: Contains `Sender`, `Recipient`, `Amount`, a `Sequence` number (replay protection), an ECDSA `PublicKey`, and a cryptographic `Signature`.
- **`BlockHeader`**: Contains `PrevHash`, `MerkleRoot`, `Timestamp`, `Difficulty`, and `Nonce`.
- **`Block`**: Combines a `BlockHeader`, block `Height`, the transaction list, and the computed `Hash`.
- **Double SHA-256 Hashing**: All hashes use `SHA256(SHA256(data))`, matching Bitcoin's approach and providing protection against length-extension attacks.

### Proof-of-Work & Parallel Mining

Mining uses a concurrent Proof-of-Work algorithm. A block is valid when its hash has at least `Difficulty` leading zero hex characters. The implementation:

1. Splits the `uint32` nonce space across all available CPU cores (one goroutine per CPU).
2. Uses Go's `context.WithCancel` so all workers stop as soon as any one finds a valid nonce.
3. **ExtraNonce for Infinite Search Space**: If the entire `uint32` nonce space (~4.3 billion values) is exhausted, the coinbase transaction's signature field is incremented (`extraNonce++`) and the Merkle Root is recalculated, creating a new effective search space. This repeats until a valid hash is found.

### Dynamic Difficulty Retargeting

After every `RetargetWindow` blocks, the chain automatically adjusts the mining difficulty to maintain the `TargetBlockTimeSec` goal:

- If blocks were mined **faster than half** the target time → difficulty **increases by 1**.
- If blocks were mined **slower than double** the target time → difficulty **decreases by 1**.
- Difficulty is always clamped to `[MinDifficulty, MaxDifficulty]`.
- The `validate` command replays the retargeting logic to ensure every block in the chain used the correct difficulty.

### Digital Signatures (ECDSA)

Transactions are signed using **ECDSA on the P-256 (secp256r1) curve**. Key features:

- **Low-S Normalization**: Both signing and verification enforce that the S component of the signature satisfies `s <= N/2`. Signatures with a high-S value are rejected, eliminating signature malleability.
- **Fixed 64-byte Signature**: R and S are each zero-padded to exactly 32 bytes, giving a canonical 64-byte signature format.
- **Address Binding**: Verification confirms that the embedded `PublicKey` actually hashes to the `Sender` address, preventing key substitution attacks.

### Merkle Tree

Transaction integrity is committed via a Merkle tree:

- Each transaction is individually hashed (double SHA-256, including `Sender`, `Recipient`, `Amount`, `PublicKey`, `Signature`).
- Leaf pairs are concatenated with a type-byte prefix (`\x00` for leaves, `\x01` for internal nodes) before hashing, preventing second-preimage attacks.
- Odd-length levels promote the unpaired node unchanged (Bitcoin-compatible).
- Only the fixed-length **Merkle Root** is stored in the block header.

### Account-Based Ledger with Replay Protection

The simulator uses an **account-based ledger** (not UTXO):

- Balances are computed by replaying all transactions from the genesis block.
- **`CalculateAvailableBalances`** deducts pending pool transactions from confirmed balances to prevent double-spending before a block is mined.
- **Sequence Numbers**: Every non-system transaction carries a monotonically increasing `Sequence` number per sender (similar to an Ethereum nonce). The chain rejects any transaction whose sequence is not exactly `lastConfirmedSequence + 1`, preventing replay attacks.

### System Addresses & Faucet

Two reserved system addresses bypass normal signature validation:

- **`COINBASE`**: Created automatically by the miner as the first transaction of every block, awarding the fixed `MiningReward` of **50 coins** to `"Miner"`.
- **`FAUCET`**: A test-only dispenser. Maximum **1,000 coins per request** and a **5,000 coin lifetime cap per address**. These limits are enforced both at submission time and during full chain validation.

### Full Chain Validation

`validate` performs a complete cryptographic audit of the chain:

1. **Genesis block**: Verifies fixed hash, Merkle Root, and zero difficulty.
2. **Each subsequent block**: Verifies height continuity, timestamp ordering, hash correctness, Merkle Root, previous hash linkage, PoW target, and the expected retargeted difficulty.
3. **Each transaction**: Verifies ECDSA signatures, sequence numbers, balance solvency, faucet limits, and COINBASE rules (first tx only, correct reward amount).

### Wallet & Keystore

- A wallet is an ECDSA P-256 key pair. The **address** is `SHA-256(compressed_public_key)` encoded as a hex string.
- The keystore (`data/wallet.json`) is a JSON map of `name -> DER-encoded private key`, supporting multiple named wallets in a single file.

### Data Persistence (Atomic Writes)

The chain is serialized to `data/chain.json` as human-readable JSON using an **atomic write** pattern (write to `.tmp`, then `os.Rename`), preventing corruption on crash. The file is loaded and validated on startup.

## Running Tests

```bash
go test ./...
```

Tests cover:

- **`internal/block`**: Merkle tree correctness (including edge cases), block hashing, PoW mining, ECDSA signing & verification.
- **`internal/chain`**: Chain creation, transaction submission, mining, difficulty retargeting, and full `Validate()` scenarios (tampered blocks, double-spends, invalid signatures, bad sequences, faucet limits).
- **`internal/ledger`**: Balance calculation and transaction validation logic.
- **`internal/wallet`**: Key generation, keystore save/load round-trips.
- **`internal/storage`**: JSON persistence.
- **`internal/cli`**: Basic CLI smoke tests.

## Known Limitations

- **No Peer Network**: The simulator is a single local process. There is no P2P networking, peer discovery, or consensus across multiple nodes — no longest-chain rule applies.
- **Static Block Reward**: The mining reward is hardcoded at 50 coins to `"Miner"` with no halving schedule or transaction fee model.
- **ExtraNonce Location**: In production blockchains (e.g., Bitcoin), the extra nonce lives inside the Coinbase script field. Here it is stored in the Coinbase transaction's `Signature` field for simplicity.
- **No Mempool Eviction**: The pending transaction pool has no size limit, expiry, or fee-based prioritization.
