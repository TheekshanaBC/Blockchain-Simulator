# start-cluster.ps1
# This script launches a 3-node local cluster of the Blockchain Simulator for testing.

Write-Host "Starting Valence Blockchain Simulator Cluster..."

# Ensure data directories exist
New-Item -ItemType Directory -Force -Path .\data\nodeA | Out-Null
New-Item -ItemType Directory -Force -Path .\data\nodeB | Out-Null
New-Item -ItemType Directory -Force -Path .\data\nodeC | Out-Null

Write-Host "Starting Node A on port 8080..."
Start-Process -FilePath "go" -ArgumentList "run ./cmd/valenced -port 8080 -data-dir ./data/nodeA -peers localhost:8081,localhost:8082" -WindowStyle Normal

Start-Sleep -Seconds 2

Write-Host "Starting Node B on port 8081..."
Start-Process -FilePath "go" -ArgumentList "run ./cmd/valenced -port 8081 -data-dir ./data/nodeB -peers localhost:8080,localhost:8082" -WindowStyle Normal

Start-Sleep -Seconds 2

Write-Host "Starting Node C on port 8082..."
Start-Process -FilePath "go" -ArgumentList "run ./cmd/valenced -port 8082 -data-dir ./data/nodeC -peers localhost:8080,localhost:8081" -WindowStyle Normal

Write-Host ""
Write-Host "Cluster is running in separate windows!"
Write-Host "Node A: http://localhost:8080"
Write-Host "Node B: http://localhost:8081"
Write-Host "Node C: http://localhost:8082"
Write-Host ""
Write-Host "Try requesting faucet funds on Node A:"
Write-Host "go run ./cmd/valence-cli -node http://localhost:8080 faucet 100"
Write-Host ""
Write-Host "Then mine a block on Node A:"
Write-Host "go run ./cmd/valence-cli -node http://localhost:8080 generate"
Write-Host ""
Write-Host "Check Node B's chain to see it propagate:"
Write-Host "go run ./cmd/valence-cli -node http://localhost:8081 getnetworkinfo"
