# Valence Complete Run Guide

Welcome to the comprehensive guide for running the Valence Blockchain Simulator. This guide walks you through launching a local cluster, generating wallets, obtaining initial funds from the Faucet, mining blocks, and sending transactions between peers.

## 1. Starting the Network (The Node Daemons)

Valence runs on a decentralized network of nodes (the `valenced` daemon). You can start a node manually, or use the provided scripts to launch a cluster.

### Launching a Local 3-Node Cluster
The easiest way to get started is to use the cluster scripts. They will automatically launch 3 nodes on ports `8080`, `8081`, and `8082` and connect them as peers.

**Windows:**
```powershell
.\start-cluster.ps1
```

**Mac/Linux:**
```bash
./start-cluster.sh
```

### Running a Node Manually
If you want to start a node manually, use the `valenced` command.

> [!IMPORTANT]
> **Faucet Keys**
> If you want a node to act as the network Faucet (allowing users to request free VCN), you **must** start it with the `--faucet-key` flag and the authorized private key. If you omit this, the node's `/faucet` API will be disabled and return `501 Not Implemented`.

**Start the Seed Node (with Faucet enabled):**
```bash
go run ./cmd/valenced -port 8080 -data-dir ./data/nodeA -faucet-key AdUl1LWR0NtSPlR6NktiYVptv2sKOwAZ8djfTt9u1Mk=
```

**Start a Peer Node (Connecting to Seed):**
```bash
go run ./cmd/valenced -port 8081 -data-dir ./data/nodeB -peers localhost:8080
```

> [!NOTE]
> **Official Development Faucet Keys**
> * **Private Key (Base64):** `AdUl1LWR0NtSPlR6NktiYVptv2sKOwAZ8djfTt9u1Mk=`
> * **Public Key (Base64):** `2z1/+FPTIcPZnfkOXM3IYypZwRLcXwgFJsOvQUrQyno=`
> * **Public Address (Hex):** `b2be6b76fa3f8e9d88de9128285f73b1deb13e8e1bd44df24e5423fce0171607`
> * **Genesis Balance:** 1,000,000,000 VCN (1 Billion VCN)

---

## 2. Setting Up Wallets (The CLI Client)

Once your nodes are running, use the `valence-cli` to interact with them. The CLI is stateless; you must tell it which node to talk to (e.g., `-node http://localhost:8080`) and which local wallet to use.

**Create a wallet for Alice:**
```bash
go run ./cmd/valence-cli -wallet alice createwallet
```

**Create a wallet for Bob:**
```bash
go run ./cmd/valence-cli -wallet bob createwallet
```

*Note: Wallets are saved locally in `./data/wallets/keys.json`. Keep this file secure.*

---

## 3. Getting Funds (The Faucet)

To start sending transactions, Alice needs some initial funds (VCN). She can request them from a node running the Faucet.

1. **Request funds from the Faucet Node (Node A on port 8080):**
   ```bash
   go run ./cmd/valence-cli -wallet alice -node http://localhost:8080 faucet 100
   ```
2. **Mine a block to confirm the Faucet transaction:**
   Transactions stay in the mempool until they are mined. Ask Node A to mine a block:
   ```bash
   go run ./cmd/valence-cli -node http://localhost:8080 generate
   ```
3. **Check Alice's Balance:**
   ```bash
   go run ./cmd/valence-cli -wallet alice -node http://localhost:8080 getbalance
   ```

---

## 4. Sending Transactions

Now that Alice has 100 VCN, she can send some to Bob.

1. **Send 25 VCN from Alice to Bob:**
   ```bash
   # Make sure to replace <bob_address> with Bob's actual public address!
   go run ./cmd/valence-cli -wallet alice -node http://localhost:8080 sendtoaddress <bob_address> 25
   ```
2. **Mine a block to confirm the transfer:**
   ```bash
   go run ./cmd/valence-cli -node http://localhost:8080 generate
   ```
3. **Verify balances:**
   ```bash
   go run ./cmd/valence-cli -wallet alice -node http://localhost:8080 getbalance
   go run ./cmd/valence-cli -wallet bob -node http://localhost:8080 getbalance
   ```

---

## 5. Network Sync & Administration

Because you have multiple nodes running, any block mined on Node A (8080) is gossiped to Node B (8081) and Node C (8082). 

**Check Node B's status:**
```bash
go run ./cmd/valence-cli -node http://localhost:8081 getnetworkinfo
```
*You should see that Node B has automatically synced the blocks you mined on Node A!*

**View pending transactions (Mempool):**
```bash
go run ./cmd/valence-cli -node http://localhost:8080 getmempoolinfo
```

**View connected peers:**
```bash
go run ./cmd/valence-cli -node http://localhost:8080 getpeerinfo
```

> [!TIP]
> **Stateless CLI**
> The `valence-cli` can point to *any* node in the network to execute a transaction. If you point it to Node C (`-node http://localhost:8082`), Node C will validate the transaction and gossip it to Nodes A and B instantly.
