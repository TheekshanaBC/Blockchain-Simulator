package gossip

import (
	"sync"
	"time"
)

// SeenCache keeps track of recently seen transaction IDs and block hashes
// to prevent infinite gossip loops.
type SeenCache struct {
	mu    sync.RWMutex
	items map[string]time.Time
	ttl   time.Duration
}

// NewSeenCache initializes a thread-safe cache for deduplication.
func NewSeenCache(ttl time.Duration) *SeenCache {
	return &SeenCache{
		items: make(map[string]time.Time),
		ttl:   ttl,
	}
}

// Add marks an item as seen.
func (c *SeenCache) Add(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[id] = time.Now()
}

// Has checks if an item has been seen. It does not update the timestamp.
func (c *SeenCache) Has(id string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.items[id]
	return exists
}

// AddIfNotSeen checks if an item is seen. If not, it adds it and returns true (meaning "was added").
// If it was already seen, it returns false. This is done atomically.
func (c *SeenCache) AddIfNotSeen(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.items[id]; exists {
		return false
	}
	c.items[id] = time.Now()
	return true
}

// PurgeOldItems removes items older than the TTL.
// This should be called periodically by the GossipEngine.
func (c *SeenCache) PurgeOldItems() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-c.ttl)
	for id, timestamp := range c.items {
		if timestamp.Before(cutoff) {
			delete(c.items, id)
		}
	}
}
