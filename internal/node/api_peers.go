package node

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
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

	cleanAddr := strings.TrimPrefix(req.Address, "http://")
	cleanAddr = strings.TrimPrefix(cleanAddr, "https://")
	if _, _, err := net.SplitHostPort(cleanAddr); err != nil {
		respondError(w, http.StatusBadRequest, "invalid peer address format")
		return
	}

	n.PeerManager.AddPeer(req.Address)
	n.PeerManager.MarkSeen(req.Address)

	for _, p := range req.Peers {
		cleanP := strings.TrimPrefix(p, "http://")
		cleanP = strings.TrimPrefix(cleanP, "https://")
		if _, _, err := net.SplitHostPort(cleanP); err == nil {
			isNew := n.PeerManager.AddPeer(p)
			if isNew {
				// Asynchronously introduce ourselves to this newly discovered peer
				go n.announceToPeer(p)
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"peers": n.PeerManager.GetPeers(),
	})
}
