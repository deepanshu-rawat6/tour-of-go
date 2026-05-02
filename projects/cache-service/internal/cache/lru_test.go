package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"tour_of_go/projects/cache-service/internal/cache"
)

func TestLRU_BasicGetSet(t *testing.T) {
	c := cache.NewLRU(3)
	defer c.Close()

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)

	v, ok := c.Get("a")
	if !ok || v != 1 {
		t.Fatalf("want 1, got %v ok=%v", v, ok)
	}
	if c.Len() != 2 {
		t.Fatalf("want len 2, got %d", c.Len())
	}
}

func TestLRU_EvictsLRU(t *testing.T) {
	c := cache.NewLRU(3)
	defer c.Close()

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)
	// Access "a" to make it recently used; "b" becomes LRU
	c.Get("a")
	c.Get("c")
	// Adding "d" should evict "b" (LRU)
	c.Set("d", 4, 0)

	if _, ok := c.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected 'a' to still be present")
	}
}

func TestLRU_TTLExpiry(t *testing.T) {
	c := cache.NewLRU(10)
	defer c.Close()

	c.Set("x", "val", 50*time.Millisecond)
	if _, ok := c.Get("x"); !ok {
		t.Fatal("expected hit before expiry")
	}
	time.Sleep(100 * time.Millisecond)
	if _, ok := c.Get("x"); ok {
		t.Fatal("expected miss after expiry")
	}
}

func TestLRU_Delete(t *testing.T) {
	c := cache.NewLRU(5)
	defer c.Close()

	c.Set("k", "v", 0)
	c.Delete("k")
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss after delete")
	}
}

func TestLRU_ConcurrentAccess(t *testing.T) {
	c := cache.NewLRU(100)
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("k%d", i)
			c.Set(key, i, time.Second)
			c.Get(key)
			c.Delete(key)
		}(i)
	}
	wg.Wait()
}
