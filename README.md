<div align="center">
  <img src="./assets/logo.png" alt="Valence Logo" width="300"/>
  <br/>
  <h3>An Educational Blockchain Simulator</h3>
  <p>A fully networked P2P node featuring Proof-of-Work, Gossip protocols, and Ed25519 cryptography.</p>
</div>

---

## ⚡ Overview

**Valence** is a clean, dependency-minimal blockchain simulator written entirely in Go. Designed primarily as an educational tool, it demonstrates the core mechanics of a decentralized network without the overwhelming complexity of production clients like `geth` or `bitcoin-core`.

By running Valence, you can inspect real-time P2P gossip, witness blockchain reorganizations (fork resolution), and interact with a live memory pool.

🌐 **Try the Web UI**: You can interact with the network via the [Valence Web App](https://valenceblockchain.vercel.app) to explore blocks, view the network topology, and manage wallets.

## 🚀 Key Features

- **Peer-to-Peer Networking**: Decentralized HTTP-based gossip engine. Nodes automatically discover peers and broadcast transactions and blocks.
- **Proof-of-Work Consensus**: Real Nakamoto consensus with dynamic difficulty retargeting based on block time.
- **Robust Cryptography**: Uses modern `Ed25519` for deterministic and secure transaction signatures.
- **Automatic Chain Syncing**: New nodes dynamically query the network, locate the tallest chain, and sync state automatically.
- **Fork Resolution**: Handles chain reorganizations by seamlessly switching to the longest valid chain and returning orphaned transactions to the mempool.
- **Concurrent Mining**: State is protected by Read/Write Mutexes, allowing nodes to continue gossiping and syncing while Proof-of-Work hashes in the background.

---

## 🛠️ Quick Start

### Prerequisites
- **Go 1.22+**

### Launching a Local Cluster

You can easily launch a 3-node cluster locally using the provided launch scripts or Docker. Each node stores its own isolated state (`chain.json` and `keystore.json`) and connects via local ports.

**Using Docker Compose (Recommended):**
```bash
docker-compose up --build
```
> **Note:** The containers map their internal ports directly to your host machine. This means your CLI, frontend UI, and web browser can still seamlessly connect to `http://localhost:8080` just as they did with the native scripts!

**Using native scripts (Windows):**
```powershell
.\start-cluster.ps1
```

**Using native scripts (Linux / macOS):**
```bash
./start-cluster.sh
```

### Running a Manual Node

To spin up a single node manually on port `8080`:
```bash
go run ./cmd/valenced -port 8080 -data-dir ./data/nodeA -peers localhost:8081 -faucet-key <base64_key>
```

---

## 💻 The Universal CLI: Performing a Transaction

Valence comes with a powerful command-line interface. Here is the complete end-to-end workflow to perform a transaction on your local cluster:

```bash
# 1. Create two wallets (Sender and Recipient)
go run ./cmd/valence-cli wallet create -name alice
go run ./cmd/valence-cli wallet create -name bob

# 2. Get some free VCN for Alice from Node A's Faucet
go run ./cmd/valence-cli -node http://localhost:8080 -wallet alice faucet 100

# 3. Mine a block on Node A to confirm the faucet transaction
go run ./cmd/valence-cli -node http://localhost:8080 generate

# 4. Verify Alice's balance
go run ./cmd/valence-cli -node http://localhost:8080 -wallet alice getbalance

# 5. Send 25 VCN from Alice to Bob via Node B (gossip in action!)
go run ./cmd/valence-cli -node http://localhost:8081 -wallet alice send -to bob -amount 25

# 6. Mine another block (on any node) to confirm the transfer
go run ./cmd/valence-cli -node http://localhost:8082 generate

# 7. Check Bob's balance!
go run ./cmd/valence-cli -node http://localhost:8080 -wallet bob getbalance
```
> **📖 Note:** See the [CLI Guide](docs/cli_guide.md) for a full breakdown of all available commands.

---

## 🏗️ Architecture Highlights

### 1. Account-Based Ledger
Unlike UTXO models, Valence uses an account-based ledger (similar to Ethereum). Sequence numbers (nonces) strictly enforce transaction ordering and prevent replay attacks, while the mempool automatically deducts pending balances to prevent double-spending.

### 2. Gossip Deduplication (`SeenCache`)
To prevent infinite broadcast storms, the P2P engine utilizes a strict `SeenCache`. Every broadcasted block hash and transaction ID is tracked globally, preventing nodes from re-forwarding items they have recently encountered.

### 3. Atomic Persistence
Chain state is persisted to disk using atomic rename operations. If a node crashes mid-write, the blockchain file (`chain.json`) remains completely uncorrupted.

---

## 📡 HTTP API Reference

Valence exposes a clean REST API for deep introspection.

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/status` | Node status, height, peer count, and mempool size. |
| `GET` | `/chain/height` | Current blockchain height. |
| `GET` | `/chain` | Fetch the full blockchain state (limited to 100k blocks). |
| `GET` | `/chain/blocks/{h}`| Get a block at a specific height. |
| `GET` | `/balances` | Confirmed balances of all accounts. |
| `GET` | `/sequence/{addr}`| Next expected sequence number (nonce) for an address. |
| `GET` | `/mempool` | List of all pending, unconfirmed transactions. |
| `GET` | `/peers` | List of all healthy connected peers. |
| `POST` | `/mine` | Force the node to mine a new block from the mempool. |
| `POST` | `/faucet` | Request funds (requires faucet key on node startup). |
| `POST` | `/tx/submit` | Submit a signed transaction. Broadcasts to peers. |

*(Internal network endpoints like `/tx/gossip`, `/block/gossip`, and `/peers/announce` are also exposed but intended strictly for node-to-node communication).*

---

## 🧪 Testing

Valence is rigorously tested for concurrency safety and cryptographic integrity. The test suite runs with the Go race detector enabled.

```bash
go test -race ./... -timeout 120s
```
