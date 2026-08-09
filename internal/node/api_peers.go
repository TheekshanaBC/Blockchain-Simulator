package node

import (
	"encoding/json"
	"net"
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
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if _, _, err := net.SplitHostPort(req.Address); err != nil {
		respondError(w, http.StatusBadRequest, "invalid peer address format")
		return
	}

	n.PeerManager.AddPeer(req.Address)
	n.PeerManager.MarkSeen(req.Address)

	for _, p := range req.Peers {
		if _, _, err := net.SplitHostPort(p); err == nil {
			n.PeerManager.AddPeer(p)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"peers": n.PeerManager.GetPeers(),
	})
}
