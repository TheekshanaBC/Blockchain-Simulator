# Phase 3: Gossip Protocol

Phase 3 introduced the decentralized P2P networking layer. The Gossip Protocol allows nodes to broadcast transactions and blocks to each other efficiently without infinite routing loops.

## 1. Gossip Engine Architecture

The `Gossip Engine` (`internal/gossip/engine.go`) is responsible for communicating with peers over HTTP. 

```mermaid
graph LR
    NodeA["Node A"] -->|"POST /tx/gossip"| NodeB["Node B"]
    NodeA -->|"POST /tx/gossip"| NodeC["Node C"]
    
    NodeB -->|"POST /tx/gossip"| NodeD["Node D"]
    NodeC -->|"POST /tx/gossip"| NodeD
```

To prevent a transaction from looping endlessly across the network, the engine utilizes a `SeenCache` (`internal/gossip/seen.go`).

## 2. Gossip Broadcast Flow

When a node receives a new transaction (via client submission) or mines a new block, it asynchronously forwards it to all known healthy peers.

```mermaid
sequenceDiagram
    participant API as HTTP API
    participant Mempool
    participant Gossip as Gossip Engine
    participant SeenCache
    participant PeerMgr as Peer Manager
    participant Network as HTTP Client

    API->>API: Receive Tx
    API->>Mempool: Add Tx
    API->>Gossip: BroadcastTx(Tx)
    
    Gossip->>SeenCache: AddIfNotSeen(Tx.ID)
    alt Already Seen
        SeenCache-->>Gossip: false (Abort Broadcast)
    else Not Seen
        SeenCache-->>Gossip: true
        Gossip->>PeerMgr: GetPeers()
        PeerMgr-->>Gossip: [Peer1, Peer2]
        
        par Async Broadcast
            Gossip->>Network: POST /tx/gossip to Peer1
            Gossip->>Network: POST /tx/gossip to Peer2
        end
    end
```

## 3. Inbound Gossip Flow

When a node *receives* gossiped data from another peer, it must validate the data thoroughly.

```mermaid
sequenceDiagram
    participant Peer as Peer Node
    participant Node API as Local Node
    participant SeenCache
    participant Chain/Mempool
    participant Gossip as Gossip Engine

    Peer->>Node API: POST /block/gossip (Block JSON)
    Node API->>SeenCache: Check Block Hash
    
    alt Hash Already Seen
        Node API-->>Peer: 200 OK (Already Seen)
    else New Hash
        Node API->>Chain/Mempool: Validate & AddBlock()
        Chain/Mempool-->>Node API: Success
        Node API->>Gossip: BroadcastBlock (Re-broadcast to other peers)
        Node API-->>Peer: 202 Accepted
    end
```

## 4. Key Improvements in Phase 3
- **Asynchronous HTTP Calls:** Goroutines are spawned for each peer request, ensuring that one slow peer does not bottleneck the gossip engine.
- **Network Boundaries:** Signatures (Ed25519) and System transactions are strictly validated at the `/gossip` API edges before reaching the internal mempool or chain.
- **Self-Healing:** If a peer fails to respond to a gossip request, the `PeerManager` marks it as unhealthy and stops gossiping to it until it recovers.
