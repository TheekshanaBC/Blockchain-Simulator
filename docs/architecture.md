# Valence Protocol Architecture

Welcome to the Valence Protocol architecture documentation. 

To give a clear understanding of how the system evolved from a simple single-process application into a fully decentralized, asynchronous peer-to-peer network, the documentation is divided by the development phases.

Please review the following documents in order to understand the architectural decisions, constraints, and implementations at each step of the journey:

- **[Phase 0: Initial Architecture (Single Node Simulator)](./phase_0_initial_architecture.md)**
  Describes the foundational Round 1 codebase before networking was introduced.

- **[Phase 1: Refactor, Rebrand & Ed25519 Migration](./phase_1_crypto_migration.md)**
  Explains the cryptographic shift from ECDSA P-256 to Ed25519 and the decoupling of the `internal/crypto` package.

- **[Phase 2: HTTP Node, API & Node Identity](./phase_2_network_node.md)**
  Details the transition to a multi-process HTTP server, the introduction of the thread-safe Mempool, and the asynchronous mining refactor.

- **[Phase 3: Gossip Protocol](./phase_3_gossip_protocol.md)**
  Covers the decentralized P2P networking layer, the Gossip Engine, and `SeenCache` deduplication to prevent infinite routing loops.

- **[Phase 4: Chain Sync, Fork Resolution, and Concurrency](./phase_4_chain_sync_and_fork_resolution.md)**
  Introduces mechanics for new nodes to catch up, network self-healing during forks, and strict concurrency safety.

- **[Phase 5: Consensus and Security](./phase_5_consensus_and_security.md)**
  Finalizes core mechanics with the Heaviest Chain Rule, dynamic difficulty retargeting, automated peer discovery, and Merkle Proofs.
