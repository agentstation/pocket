package store

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	testValue1 = "value1"
)

func TestBoundedStore(t *testing.T) {
	t.Run("enforces max entries limit", func(t *testing.T) {
		store := NewBoundedStore(
			WithMaxEntries(3),
			WithEvictionPolicy(FIFO),
		)
		ctx := context.Background()

		// Add 4 entries (should evict the first one)
		store.Set(ctx, "key1", testValue1)
		store.Set(ctx, "key2", "value2")
		store.Set(ctx, "key3", "value3")
		store.Set(ctx, "key4", "value4")

		// First entry should be evicted
		if _, exists := store.Get(ctx, "key1"); exists {
			t.Error("key1 should have been evicted")
		}

		// Other entries should exist
		for _, key := range []string{"key2", "key3", "key4"} {
			if _, exists := store.Get(ctx, key); !exists {
				t.Errorf("%s should exist", key)
			}
		}
	})

	t.Run("LRU eviction policy", func(t *testing.T) {
		store := NewBoundedStore(
			WithMaxEntries(3),
			WithEvictionPolicy(LRU),
		)
		ctx := context.Background()

		// Add 3 entries
		store.Set(ctx, "key1", testValue1)
		store.Set(ctx, "key2", "value2")
		store.Set(ctx, "key3", "value3")

		// Access key1 and key3 (making key2 least recently used)
		store.Get(ctx, "key1")
		store.Get(ctx, "key3")

		// Add new entry (should evict key2)
		store.Set(ctx, "key4", "value4")

		if _, exists := store.Get(ctx, "key2"); exists {
			t.Error("key2 should have been evicted as LRU")
		}

		// Other keys should exist
		for _, key := range []string{"key1", "key3", "key4"} {
			if _, exists := store.Get(ctx, key); !exists {
				t.Errorf("%s should exist", key)
			}
		}
	})

	t.Run("TTL eviction", func(t *testing.T) {
		store := NewBoundedStore(
			WithTTL(50 * time.Millisecond),
		)
		ctx := context.Background()

		// Add entry
		store.Set(ctx, "key1", testValue1)

		// Should exist immediately
		if _, exists := store.Get(ctx, "key1"); !exists {
			t.Error("key1 should exist")
		}

		// Wait for TTL
		time.Sleep(100 * time.Millisecond)

		// Should be expired
		if _, exists := store.Get(ctx, "key1"); exists {
			t.Error("key1 should have expired")
		}
	})

	t.Run("eviction callback", func(t *testing.T) {
		evicted := make(map[string]any)
		store := NewBoundedStore(
			WithMaxEntries(2),
			WithEvictionCallback(func(key string, value any) {
				evicted[key] = value
			}),
		)
		ctx := context.Background()

		store.Set(ctx, "key1", testValue1)
		store.Set(ctx, "key2", "value2")
		store.Set(ctx, "key3", "value3") // Should evict key1

		if evicted["key1"] != testValue1 {
			t.Error("eviction callback not called correctly")
		}
	})

	t.Run("size limit enforcement", func(t *testing.T) {
		store := NewBoundedStore(
			WithMaxSize(100),
			WithEvictionPolicy(FIFO),
		)
		ctx := context.Background()

		// Add entries with estimated sizes
		store.Set(ctx, "key1", "short") // ~5 bytes
		store.Set(ctx, "key2", "a much longer string value") // ~26 bytes
		
		stats := store.GetStats()
		if stats.CurrentSize == 0 {
			t.Error("size tracking not working")
		}

		// Try to add value that exceeds max size
		err := store.Set(ctx, "huge", make([]byte, 200))
		if err == nil {
			t.Error("should reject value larger than max size")
		}
	})

	t.Run("scoped store", func(t *testing.T) {
		store := NewBoundedStore()
		ctx := context.Background()

		// Create scoped stores
		userStore := store.Scope("user")
		adminStore := store.Scope("admin")

		// Set values in different scopes
		_ = userStore.Set(ctx, "name", "john")
		_ = adminStore.Set(ctx, "name", "alice")

		// Values should be isolated
		userVal, _ := userStore.Get(ctx, "name")
		if userVal != "john" {
			t.Errorf("expected john, got %v", userVal)
		}

		adminVal, _ := adminStore.Get(ctx, "name")
		if adminVal != "alice" {
			t.Errorf("expected alice, got %v", adminVal)
		}

		// Base store should see scoped keys
		if _, exists := store.Get(ctx, "user:name"); !exists {
			t.Error("base store should see scoped keys")
		}
	})
}

func TestMultiTieredStore(t *testing.T) {
	t.Run("promotes values on access", func(t *testing.T) {
		// Create tiers with different characteristics
		tier1 := NewBoundedStore(WithMaxEntries(10))  // Fast tier
		tier2 := NewBoundedStore(WithMaxEntries(100)) // Slow tier

		multi := NewMultiTieredStore(tier1, tier2)
		ctx := context.Background()

		// Write to tier 2 directly
		_ = tier2.Set(ctx, "key1", testValue1)

		// Access through multi-store should promote to tier 1
		val, exists := multi.Get(ctx, "key1")
		if !exists || val != testValue1 {
			t.Error("should find value in tier 2")
		}

		// Allow promotion to complete
		time.Sleep(10 * time.Millisecond)

		// Should now exist in tier 1
		if _, exists := tier1.Get(ctx, "key1"); !exists {
			t.Error("value should be promoted to tier 1")
		}
	})

	t.Run("writes to first tier", func(t *testing.T) {
		tier1 := NewBoundedStore()
		tier2 := NewBoundedStore()

		multi := NewMultiTieredStore(tier1, tier2)
		ctx := context.Background()

		// Write through multi-store
		_ = multi.Set(ctx, "key1", testValue1)

		// Should exist in tier 1
		if _, exists := tier1.Get(ctx, "key1"); !exists {
			t.Error("value should be in tier 1")
		}

		// Should not exist in tier 2
		if _, exists := tier2.Get(ctx, "key1"); exists {
			t.Error("value should not be in tier 2 initially")
		}
	})

	t.Run("deletes from all tiers", func(t *testing.T) {
		tier1 := NewBoundedStore()
		tier2 := NewBoundedStore()

		multi := NewMultiTieredStore(tier1, tier2)
		ctx := context.Background()

		// Add to both tiers
		_ = tier1.Set(ctx, "key1", testValue1)
		_ = tier2.Set(ctx, "key1", testValue1)

		// Delete through multi-store
		_ = multi.Delete(ctx, "key1")

		// Should be deleted from both
		if _, exists := tier1.Get(ctx, "key1"); exists {
			t.Error("key should be deleted from tier 1")
		}
		if _, exists := tier2.Get(ctx, "key1"); exists {
			t.Error("key should be deleted from tier 2")
		}
	})
}

func TestShardedStore(t *testing.T) {
	t.Run("distributes keys across shards", func(t *testing.T) {
		store := NewShardedStore(4)
		ctx := context.Background()

		// Add many keys
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key%d", i)
			store.Set(ctx, key, i)
		}

		// All keys should be retrievable
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key%d", i)
			val, exists := store.Get(ctx, key)
			if !exists {
				t.Errorf("key %s should exist", key)
			}
			if val != i {
				t.Errorf("expected %d, got %v", i, val)
			}
		}
	})

	t.Run("scoped sharded store", func(t *testing.T) {
		store := NewShardedStore(4)
		ctx := context.Background()

		scoped := store.Scope("prefix")
		_ = scoped.Set(ctx, "key1", testValue1)

		// Should be accessible through scoped store
		val, exists := scoped.Get(ctx, "key1")
		if !exists || val != testValue1 {
			t.Error("scoped value not found")
		}

		// Should not be accessible without prefix
		if _, exists := store.Get(ctx, "key1"); exists {
			t.Error("should not find unprefixed key")
		}

		// Should be accessible with full key
		if _, exists := store.Get(ctx, "prefix:key1"); !exists {
			t.Error("should find prefixed key")
		}
	})
}