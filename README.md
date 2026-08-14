# Blockchain Simulator (Round 2 - Networked Node)

An educational blockchain simulator written in Go, demonstrating the core mechanics of a production-quality blockchain. In Round 2, the project has evolved from a local CLI tool into a fully functional **Peer-to-Peer Networked Node** with HTTP APIs, Gossip protocols, Chain Synchronization, and Fork Resolution.


## What Changed from Round 1?

- **HTTP Node Architecture**: The `cmd/blockchainsimulator` CLI was removed. The application now runs as a long-lived HTTP daemon (`cmd/valence-node`).
- **Ed25519 Cryptography**: Migrated from ECDSA (secp256r1) to Ed25519 for simpler, faster, and more robust deterministic signatures.
- **Gossip Protocol**: Added a P2P gossip engine (`internal/gossip`) with duplicate detection (`SeenCache`) to broadcast transactions and mined blocks across the network.
- **Chain Synchronization**: New nodes can join the network and automatically download the full chain from the highest peer (`internal/sync`).
- **Fork Resolution & Reorg**: If two nodes mine at the same height, they resolve the fork by adopting the longest valid chain, returning any orphaned transactions to the mempool (`chain.SwitchToChain`).
- **Concurrency**: State is now protected by `sync.RWMutex`. Mining was decoupled from the chain lock, allowing nodes to continue gossiping and syncing while Proof-of-Work runs concurrently.

---

## Prerequisites

- **Go 1.22** or newer

---

## Project Structure

```text
blockchain-simulator/
+-- cmd/
¦   +-- mining_experiment/  # Standalone PoW benchmarking tool
   +-- valence-cli/        # The Universal Client (Wallet & Admin Tool)
   +-- valenced/           # MAIN ENTRY: The Networked Blockchain Node Daemon
+-- internal/
¦   +-- block/              # Block, Transaction, Merkle, Ed25519 Signing, PoW Mining
¦   +-- chain/              # Chain state, validation, reorg, difficulty, faucet
¦   +-- crypto/             # Ed25519 wrappers and address generation
¦   +-- gossip/             # P2P broadcast engine & SeenCache deduplication
¦   +-- ledger/             # Account balances & sequence (nonce) tracking
¦   +-- node/               # HTTP Server, Mempool, API Endpoints
¦   +-- peer/               # Peer manager & health tracking
¦   +-- storage/            # JSON persistence (atomic writes)
¦   +-- sync/               # Chain synchronization logic
¦   +-- wallet/             # Key generation & keystore
+-- data/                   # Runtime data per node (chain.json, keystore.json)
```

---

## Starting a Local Cluster

You can easily launch a 3-node cluster locally using the provided launch scripts. 

**On Windows:**
```powershell
.\start-cluster.ps1
```

**On Linux / Mac (Bash):**
```bash
./start-cluster.sh
```

This will launch three nodes on ports `8080`, `8081`, and `8082`. Each node stores its own isolated state in `./data/nodeA`, `./data/nodeB`, and `./data/nodeC`. They will automatically connect to each other via the `-peers` flag.

### Running a Node Manually

```bash
go run ./cmd/valenced -port 8080 -data-dir ./data/nodeA -peers localhost:8081,localhost:8082 -faucet-key <base64_key>
```

### The Client CLI (`valence-cli`)

Instead of raw `curl` commands, you can manage wallets and interact with the nodes using our unified `valence-cli` tool.

**Full Guide:** See [docs/cli_guide.md](./docs/cli_guide.md) for detailed instructions.

```bash
# Example: Check Alice's balance on Node B
go run ./cmd/valence-cli -node http://localhost:8081 -wallet alice getbalance
```

#### Command-Line Flags

| Flag | Default | Description |
|---|---|---|
| `-port` | `3001` | HTTP port to listen on |
| `-data-dir` | `./data/node1` | Directory to store `chain.json` and `keystore.json` |
| `-peers` | `""` | Comma-separated list of peer addresses to connect to on startup |
| `-difficulty` | `3` | Initial mining difficulty |
| `-retarget-window` | `4` | Number of blocks between difficulty retargets |
| `-target-block-time`| `10` | Target time per block in seconds |
| `-faucet-key` | `""` | Base64-encoded private key to enable the Faucet on this node |

---

## HTTP API Reference

The node exposes a JSON HTTP API. You can interact with it using `curl` or Postman.

### Introspection Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/status` | Node status (height, head hash, peer count, mempool size, node address). |
| `GET` | `/chain/height` | Current blockchain height. |
| `GET` | `/chain` | Full blockchain (limited to 100k blocks). |
| `GET` | `/chain/blocks/{h}`| Get block at specific height. |
| `GET` | `/balances` | Confirmed balances of all accounts. |
| `GET` | `/sequence/{addr}` | Next expected sequence number (nonce) for an address. |
| `GET` | `/mempool` | List of all pending transactions. |
| `GET` | `/peers` | List of all healthy connected peers. |

### Action Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/faucet` | Request up to 1000 VCN. Body: `{"address": "...", "amount": 100}` |
| `POST` | `/mine` | Mine a new block from the mempool. |
| `POST` | `/tx/submit` | Submit a signed transaction. Broadcasts to peers. |

### Network Endpoints (Used by Nodes internally)

| Method | Path | Description |
|---|---|---|
| `POST` | `/tx/gossip` | Receive a gossiped transaction from a peer. |
| `POST` | `/block/gossip`| Receive a mined block from a peer. |
| `POST` | `/peers/announce`| Register a new peer and exchange peer lists. |

---

## Architecture & Design Decisions

### 1. Gossip Deduplication (`SeenCache`)
To prevent infinite broadcast loops, `internal/gossip` uses a `SeenCache`. Every broadcasted block hash and transaction ID is tracked. The cache prevents re-forwarding items we've recently seen. The Mempool provides a secondary layer of deduplication by rejecting duplicates at the API edge.

### 2. Proof-of-Work & Concurrency
Proof-of-Work mining is computationally intensive. In Round 2, `MineBlock` was decoupled from the main Chain `sync.RWMutex`. 
1. The node acquires a read lock to validate transactions and build the block header.
2. It **releases the lock** and performs PoW locally.
3. Once the hash is found, it calls `AddBlock`, which re-acquires a write lock, validates the block, and appends it.
This allows the node to continue handling gossip and sync requests while mining.

### 3. Chain Synchronization & Fork Resolution
If a node receives a block via gossip that is further ahead than its current height + 1, it triggers `SyncFromBestPeer()`. 
The node queries all known peers, finds the tallest chain, downloads it, and calls `chain.SwitchToChain()`. 
`SwitchToChain` validates the entire candidate chain from Genesis. If valid, it swaps out the local chain, finds any "orphaned" transactions from the abandoned fork, and returns them to the mempool to be re-mined.

### 4. Account-Based Ledger with Replay Protection
The simulator uses an account-based ledger (like Ethereum). 
- **`CalculateAvailableBalances`** deducts pending mempool transactions from confirmed balances to prevent double-spending attacks.
- **Sequence Numbers**: Every transaction carries a strict, monotonic sequence number per sender, preventing replay attacks.

### 5. Data Persistence
State is persisted to `./data/.../chain.json` using atomic writes (writing to a `.tmp` file and running `os.Rename`). This ensures that if the node crashes mid-write, the blockchain file is not corrupted.

---

## Running Tests

The test suite runs with the Go race detector enabled to ensure complete concurrency safety.

```bash
go test -race ./... -timeout 120s
```

Tests cover cryptographic signing, gossip deduplication, chain reorg logic, mempool race safety, peer management, and complete validation of tampered state (invalid signatures, broken merkle roots, fake difficulty timestamps).

