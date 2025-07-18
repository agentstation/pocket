package pocket_test

import (
	"context"
	"sync"
	"testing"
	
	"github.com/agentstation/pocket"
)

func TestStoreConcurrency(t *testing.T) {
	store := pocket.NewStore()
	var wg sync.WaitGroup
	
	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('a' + n%26))
			store.Set(key, n)
		}(i)
	}
	
	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('a' + n%26))
			store.Get(key)
		}(i)
	}
	
	wg.Wait()
	
	// Verify some values
	val, ok := store.Get("a")
	if !ok {
		t.Error("Expected value for key 'a'")
	}
	if val.(int)%26 != 0 {
		t.Error("Unexpected value modulo")
	}
}

func TestTypedStore(t *testing.T) {
	type User struct {
		ID   string
		Name string
		Age  int
	}
	
	store := pocket.NewStore()
	userStore := pocket.NewTypedStore[User](store)
	ctx := context.Background()
	
	tests := []struct {
		name    string
		op      func() error
		check   func(t *testing.T)
		wantErr bool
	}{
		{
			name: "set and get user",
			op: func() error {
				user := User{ID: "123", Name: "Alice", Age: 30}
				return userStore.Set(ctx, "user:123", user)
			},
			check: func(t *testing.T) {
				user, exists, err := userStore.Get(ctx, "user:123")
				if err != nil {
					t.Errorf("Get() error = %v", err)
				}
				if !exists {
					t.Error("Get() exists = false, want true")
				}
				if user.Name != "Alice" {
					t.Errorf("Get() user.Name = %v, want Alice", user.Name)
				}
			},
		},
		{
			name: "get non-existent key",
			op:   func() error { return nil },
			check: func(t *testing.T) {
				_, exists, err := userStore.Get(ctx, "user:999")
				if err != nil {
					t.Errorf("Get() error = %v", err)
				}
				if exists {
					t.Error("Get() exists = true, want false")
				}
			},
		},
		{
			name: "type mismatch",
			op: func() error {
				// Store a different type
				store.Set("user:bad", "not a user")
				return nil
			},
			check: func(t *testing.T) {
				_, _, err := userStore.Get(ctx, "user:bad")
				if err == nil {
					t.Error("Get() error = nil, want type error")
				}
			},
		},
		{
			name: "delete user",
			op: func() error {
				user := User{ID: "456", Name: "Bob", Age: 25}
				userStore.Set(ctx, "user:456", user)
				return userStore.Delete(ctx, "user:456")
			},
			check: func(t *testing.T) {
				_, exists, err := userStore.Get(ctx, "user:456")
				if err != nil {
					t.Errorf("Get() error = %v", err)
				}
				if exists {
					t.Error("Get() after delete exists = true, want false")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op()
			if (err != nil) != tt.wantErr {
				t.Errorf("op() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.check(t)
		})
	}
}

func TestScopedStore(t *testing.T) {
	baseStore := pocket.NewStore()
	userStore := pocket.NewScopedStore(baseStore, "user")
	adminStore := pocket.NewScopedStore(baseStore, "admin")
	
	// Set values in different scopes
	userStore.Set("name", "Alice")
	adminStore.Set("name", "Bob")
	
	// Check isolation
	userName, ok := userStore.Get("name")
	if !ok || userName != "Alice" {
		t.Errorf("userStore.Get(name) = %v, %v; want Alice, true", userName, ok)
	}
	
	adminName, ok := adminStore.Get("name")
	if !ok || adminName != "Bob" {
		t.Errorf("adminStore.Get(name) = %v, %v; want Bob, true", adminName, ok)
	}
	
	// Check that base store has prefixed keys
	userPrefixed, ok := baseStore.Get("user:name")
	if !ok || userPrefixed != "Alice" {
		t.Errorf("baseStore.Get(user:name) = %v, %v; want Alice, true", userPrefixed, ok)
	}
	
	// Test delete
	userStore.Delete("name")
	_, ok = userStore.Get("name")
	if ok {
		t.Error("userStore.Get(name) after delete returned true, want false")
	}
	
	// Admin scope should still have its value
	adminName, ok = adminStore.Get("name")
	if !ok || adminName != "Bob" {
		t.Errorf("adminStore.Get(name) after user delete = %v, %v; want Bob, true", adminName, ok)
	}
}

func BenchmarkStore(b *testing.B) {
	b.Run("Set", func(b *testing.B) {
		store := pocket.NewStore()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			store.Set("key", i)
		}
	})
	
	b.Run("Get", func(b *testing.B) {
		store := pocket.NewStore()
		store.Set("key", "value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			store.Get("key")
		}
	})
	
	b.Run("Concurrent", func(b *testing.B) {
		store := pocket.NewStore()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%2 == 0 {
					store.Set("key", i)
				} else {
					store.Get("key")
				}
				i++
			}
		})
	})
}