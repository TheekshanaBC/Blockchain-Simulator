package gossip

import (
	"sync"
	"testing"
	"time"
)

/*
TestSeenCache_AddAndHas verifies the basic functionality of the cache:
adding an item and subsequently checking for its existence.
*/
func TestSeenCache_AddAndHas(t *testing.T) {
	// Initialize cache with a 1-hour TTL
	cache := NewSeenCache(1 * time.Hour)

	// 1. Verify the cache is initially empty
	if cache.Has("tx1") {
		t.Error("Expected cache to not have tx1 initially")
	}

	// 2. Add an item to the cache
	cache.Add("tx1")

	// 3. Verify the item now exists in the cache
	if !cache.Has("tx1") {
		t.Error("Expected cache to have tx1 after adding")
	}
}

/*
TestSeenCache_AddIfNotSeen tests the atomic deduplication check.
It ensures that concurrent-safe logic correctly identifies new vs existing items.
*/
func TestSeenCache_AddIfNotSeen(t *testing.T) {
	cache := NewSeenCache(1 * time.Hour)

	// 1. Adding a completely new item should succeed and return true
	added := cache.AddIfNotSeen("tx2")
	if !added {
		t.Error("Expected AddIfNotSeen to return true for new item")
	}

	// 2. Attempting to add the exact same item again should fail and return false
	addedAgain := cache.AddIfNotSeen("tx2")
	if addedAgain {
		t.Error("Expected AddIfNotSeen to return false for already seen item")
	}
}

/*
TestSeenCache_PurgeOldItems ensures that items older than the TTL
are successfully removed from the cache to prevent memory leaks.
*/
func TestSeenCache_PurgeOldItems(t *testing.T) {
	// Use a very short TTL (10ms) for testing
	cache := NewSeenCache(10 * time.Millisecond)

	// 1. Add a normal item (timestamp = now)
	cache.Add("tx1")
	
	// 2. Add another item but manipulate its timestamp directly to simulate it being old
	cache.mu.Lock()
	cache.items["tx2"] = time.Now().Add(-1 * time.Hour)
	cache.mu.Unlock()

	// 3. Run the purge mechanism
	cache.PurgeOldItems()

	// 4. Verify the fresh item (tx1) is still kept
	if !cache.Has("tx1") {
		t.Error("Expected tx1 to remain as it is not old")
	}

	// 5. Verify the old item (tx2) was correctly purged
	if cache.Has("tx2") {
		t.Error("Expected tx2 to be purged as it is older than TTL")
	}
}

/*
TestSeenCache_Concurrency runs heavy parallel operations against the cache
to ensure there are no data races or deadlocks.
*/
func TestSeenCache_Concurrency(t *testing.T) {
	cache := NewSeenCache(1 * time.Hour)
	var wg sync.WaitGroup

	// Concurrently add and check 100 items to ensure no race conditions
	for i := 0; i < 100; i++ {
		wg.Add(1)
		
		// Spawn a goroutine for each item
		go func(id string) {
			defer wg.Done()
			
			// Both operations acquire internal mutex locks
			cache.AddIfNotSeen(id)
			_ = cache.Has(id)
		}(string(rune(i)))
	}

	// Wait for all goroutines to complete
	wg.Wait()
}
