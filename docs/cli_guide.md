# Valence CLI Reference Guide

The `valence-cli` is your remote control for interacting with the Valence Blockchain Simulator. It follows a stateless architecture similar to production blockchains like Bitcoin (`bitcoin-cli`). 

> [!NOTE]
> **Understanding the Architecture**
> - **Nodes (`valenced`)**: The actual servers that run the blockchain, hold the `chain.json`, and connect to other peers.
> - **CLI (`valence-cli`)**: Your offline tool that holds your private keys, builds transactions, signs them, and sends them to a Node. The CLI shuts down immediately after finishing your command.

---

## 🛠️ Global Flags (The Basics)

Because the CLI is stateless, you often need to tell it **which node to talk to** and **which wallet to use** for a specific command. You do this using Global Flags.

| Flag | What it does | Default Value |
| :--- | :--- | :--- |
| `-node` | The URL of the node you want to connect to. | `http://localhost:8080` |
| `-keystore` | Where your private keys are saved. | `./data/wallets/keys.json` |
| `-wallet` | The specific name of the wallet to use. | `primary` |

**Example of combining flags:**
```bash
# Connect to Node B (8081) and use Bob's wallet to check his balance
go run ./cmd/valence-cli -node http://localhost:8081 -wallet bob getbalance
```

---

## 💰 Wallet Commands (For Users)

These commands are what regular users use to manage their funds.

### `createwallet`
Generates a brand new wallet (private/public key pair), saves it to your keystore file, and shows you your new public address.
```bash
go run ./cmd/valence-cli -wallet charlie createwallet
```

### `faucet <amount>`
Requests free test coins (VCN) from the network to get started. *(Limit: 1000 VCN per request).*
```bash
go run ./cmd/valence-cli -wallet charlie faucet 500
```

### `getbalance [address]`
Checks the confirmed balance on the blockchain. 
- If you don't provide an address, it checks the active wallet's balance.
```bash
go run ./cmd/valence-cli -wallet charlie getbalance
```

### `sendtoaddress <recipient_address> <amount>`
Creates a transaction, mathematically signs it with your private key, and broadcasts it to the network.
```bash
go run ./cmd/valence-cli -wallet charlie sendtoaddress <bob_address> 50
```

---

## ⚙️ Node Admin Commands (For Operators)

These commands are used to check the health of the network or force the simulator to perform actions.

### `getnetworkinfo` (or `getblockchaininfo`)
Gets the current status of the node you are connected to, including how many blocks exist (height) and the latest block hash (tip).
```bash
go run ./cmd/valence-cli getnetworkinfo
```

### `getmempoolinfo`
Shows all pending transactions that are waiting to be mined into a block.
```bash
go run ./cmd/valence-cli getmempoolinfo
```

### `generate` (Mine a Block)
Manually tells the node to mine a new block. It will grab all pending transactions from the mempool and pack them into the blockchain.
```bash
# Force Node A to mine a block
go run ./cmd/valence-cli generate
```

### `getpeerinfo`
Lists all other nodes (peers) that this node is currently talking to.
```bash
go run ./cmd/valence-cli getpeerinfo
```

### `addnode <address>`
Manually forces this node to connect to a new peer on the network.
```bash
go run ./cmd/valence-cli addnode localhost:8082
```

---

## 🚀 Step-by-Step Example Workflow

Want to see it in action? Follow these steps:

1. **Start the cluster** (Run this in a separate terminal and leave it running):
   ```bash
   ./start-cluster.ps1
   ```
2. **Create two wallets:**
   ```bash
   go run ./cmd/valence-cli -wallet alice createwallet
   go run ./cmd/valence-cli -wallet bob createwallet
   ```
3. **Get free coins for Alice:**
   ```bash
   go run ./cmd/valence-cli -wallet alice faucet 100
   go run ./cmd/valence-cli generate  # Mine the block to confirm!
   ```
4. **Send 25 coins from Alice to Bob:**
   ```bash
   go run ./cmd/valence-cli -wallet alice sendtoaddress <put_bobs_address_here> 25
   ```
5. **Confirm the transaction:**
   ```bash
   go run ./cmd/valence-cli generate  # Mine a block again
   go run ./cmd/valence-cli -wallet bob getbalance
   ```
