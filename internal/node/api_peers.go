package node

import (
	"encoding/json"
	"net/http"
)

// GET /peers
func (n *Node) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, n.PeerManager.GetPeers())
}

// POST /peers/announce
func (n *Node) handlePeersAnnounce(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address string   `json:"address"`
		Peers   []string `json:"peers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	n.PeerManager.AddPeer(req.Address)
	n.PeerManager.MarkSeen(req.Address)

	for _, p := range req.Peers {
		n.PeerManager.AddPeer(p)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"peers": n.PeerManager.GetPeers(),
	})
}
