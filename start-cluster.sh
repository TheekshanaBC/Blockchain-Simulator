#!/bin/bash
# start-cluster.sh
# Launches a 3-node local cluster of the Blockchain Simulator for testing.

echo "Starting Valence Blockchain Simulator Cluster..."

# Ensure data directories exist
mkdir -p ./data/nodeA ./data/nodeB ./data/nodeC

# Start Node A
echo "Starting Node A on port 8080..."
go run ./cmd/valence-node -port 8080 -data-dir ./data/nodeA -peers localhost:8081,localhost:8082 > ./data/nodeA/node.log 2>&1 &
NODE_A_PID=$!

sleep 2

# Start Node B
echo "Starting Node B on port 8081..."
go run ./cmd/valence-node -port 8081 -data-dir ./data/nodeB -peers localhost:8080,localhost:8082 > ./data/nodeB/node.log 2>&1 &
NODE_B_PID=$!

sleep 2

# Start Node C
echo "Starting Node C on port 8082..."
go run ./cmd/valence-node -port 8082 -data-dir ./data/nodeC -peers localhost:8080,localhost:8081 > ./data/nodeC/node.log 2>&1 &
NODE_C_PID=$!

echo "Cluster is running in the background!"
echo "Node A PID: $NODE_A_PID (logs: ./data/nodeA/node.log)"
echo "Node B PID: $NODE_B_PID (logs: ./data/nodeB/node.log)"
echo "Node C PID: $NODE_C_PID (logs: ./data/nodeC/node.log)"
echo ""
echo "To stop the cluster, run: kill $NODE_A_PID $NODE_B_PID $NODE_C_PID"
echo ""
echo "Try requesting faucet funds on Node A:"
echo "curl -X POST http://localhost:8080/faucet -H ""Content-Type: application/json"" -d '{""address"": ""test"", ""amount"": 100}'"
echo ""
echo "Then mine a block on Node A:"
echo "curl -X POST http://localhost:8080/mine"
echo ""
echo "Check Node B's chain to see it propagate:"
echo "curl http://localhost:8081/chain/height"

