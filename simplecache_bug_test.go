// SPDX-License-Identifier: MIT

// Regression tests for bugs found in review. Each test asserts the correct
// behaviour, so they all pass once the fixes are in place in simplecache.go.
// TestMaxAgeIsHardExpiryAcrossGetMethods is the exception in one respect:
// its GetWithStatus and Peek subtests pass against both the buggy and the
// fixed code on purpose, pinning the behaviour the fix must preserve.
package simplecache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// BUG 1: Set on a full cache evicts an entry even when the key already exists,
// because eviction runs before the existing-key check (simplecache.go, Set).
// An update must not remove an unrelated entry: no new space is needed.
func TestSetUpdateExistingKeyDoesNotEvict(t *testing.T) {
	mi := 3
	es := 10
	sc := New[int](Option{MaxItems: &mi, EvictionSamples: &es})

	_ = sc.Set("a", 1)
	_ = sc.Set("b", 2)
	_ = sc.Set("c", 3)

	// Give each key a distinct hit count so the LFU victim is deterministic
	// (a=0, c=5, b=10) instead of a coin flip over equal counts.
	for i := 0; i < 5; i++ {
		_, _ = sc.Get("c")
	}
	for i := 0; i < 10; i++ {
		_, _ = sc.Get("b")
	}

	// "b" already exists; updating it must not evict anything.
	_ = sc.Set("b", 20)

	assert.Equal(t, 3, sc.Sum(), "cache shrank due to an update, not an insert")

	_, ok := sc.Get("a")
	assert.True(t, ok, "'a' (the LFU victim) was evicted by an update of 'b'")

	_, ok = sc.Get("c")
	assert.True(t, ok, "'c' was evicted by an update of 'b'")

	v, ok := sc.Get("b")
	assert.True(t, ok)
	assert.Equal(t, 20, v)
}

// BUG 2: LRU evicts by creation time (entry.age), not last access (entry.lastUse),
// making it FIFO. evictLRU (simplecache.go) compares v.age, while lastUse is
// maintained in Get specifically to support LRU.
//
// "a" is created FIRST but accessed 100 times afterwards: a correct LRU must
// keep it and evict "b" or "c" instead. EvictionSamples is set higher than the
// cache size so the victim is chosen from all entries (deterministic).
func TestLRUEvictsByLastUseNotCreationTime(t *testing.T) {
	ep := LRU
	mi := 3
	es := 100
	sc := New[int](Option{MaxItems: &mi, EvictionPolicy: &ep, EvictionSamples: &es})

	_ = sc.Set("a", 1) // created first
	_ = sc.Set("b", 2)
	_ = sc.Set("c", 3)

	for i := 0; i < 100; i++ {
		_, _ = sc.Get("a") // "a" becomes the most recently USED entry
	}

	_ = sc.Set("d", 4) // cache full -> one entry must be evicted

	_, ok := sc.Get("a")
	assert.True(t, ok, "most recently USED entry 'a' was evicted: policy is FIFO (entry.age), not LRU (entry.lastUse)")
}

// BUG 3: Set resets hits to 0 even when updating an existing key, so the most
// frequently used key becomes the first LFU victim immediately after an update.
//
// Hit counts before the update: a=50, b=1, c=2. After "Set("a", 10)" the
// buggy code zeroes a. Eviction on Set("e") then picks a, the actual hot key.
// After the update a is the unique minimum hit count (0), so the LFU victim
// is deterministic.
func TestSetUpdateDoesNotResetHitCount(t *testing.T) {
	ep := LFU
	mi := 4
	es := 100
	sc := New[int](Option{MaxItems: &mi, EvictionPolicy: &ep, EvictionSamples: &es})

	_ = sc.Set("a", 1)
	_ = sc.Set("b", 2)
	_ = sc.Set("c", 3)

	for i := 0; i < 50; i++ {
		_, _ = sc.Get("a")
	}
	_, _ = sc.Get("b")
	_, _ = sc.Get("c")
	_, _ = sc.Get("c")

	_ = sc.Set("a", 10) // update only; cache is not full (3 < 4), no eviction
	_ = sc.Set("d", 4)  // now full
	_, _ = sc.Get("d")
	_ = sc.Set("e", 5) // forces one eviction

	_, ok := sc.Get("a")
	assert.True(t, ok, "hot key 'a' evicted by LFU because Set reset its hit count to 0")
}

// BUG 4: MaxAge has different semantics per read method. The field comment
// (simplecache.go: "at what point should it be evicted no matter what") and
// the README's stale-while-revalidate section treat MaxAge as a hard expiry,
// and Peek + GetWithStatus implement it from creation time (entry.age).
// The buggy Get instead measured from lastUse, so an entry that is read
// continuously never expired in Get while the other methods reported it as
// expired.
//
// Only the Get subtest needs the keepAlive loop, because only the buggy Get
// is affected by refreshing lastUse: it must still hard-expire an entry that
// is read continuously. The other two subtests just sleep past MaxAge and
// verify their own method's expiry; they pass both before and after the fix.
//
// keepAlive polls at 30ms against a 150ms MaxAge so keep-alive survives
// normal scheduling jitter. Poll success is deliberately not asserted: once
// Get is fixed to be creation-based, the polls start failing partway through,
// which is expected.
func TestMaxAgeIsHardExpiryAcrossGetMethods(t *testing.T) {
	maxAge := 150 * time.Millisecond

	keepAlive := func(sc *Cache[string], key string) {
		start := time.Now()
		for time.Since(start) < 240*time.Millisecond {
			time.Sleep(30 * time.Millisecond)
			_, _ = sc.Get(key)
		}
	}

	t.Run("Get expires past MaxAge", func(t *testing.T) {
		sc := New[string](Option{MaxAge: &maxAge})
		_ = sc.Set("k", "v")
		keepAlive(sc, "k")

		_, ok := sc.Get("k")
		assert.False(t, ok, "Get must hard-expire at creation-time MaxAge even while the entry is read continuously")
		assert.Equal(t, 0, sc.Sum(), "expired entry must be deleted")
	})

	t.Run("GetWithStatus expires past MaxAge", func(t *testing.T) {
		sc := New[string](Option{MaxAge: &maxAge})
		_ = sc.Set("k", "v")
		time.Sleep(200 * time.Millisecond)

		res := sc.GetWithStatus("k")
		assert.Equal(t, CacheMiss, res.Status, "GetWithStatus must hard-expire at creation-time MaxAge")
		assert.Equal(t, 0, sc.Sum())
	})

	t.Run("Peek reports stale past MaxAge without deleting", func(t *testing.T) {
		sc := New[string](Option{MaxAge: &maxAge})
		_ = sc.Set("k", "v")
		time.Sleep(200 * time.Millisecond)

		res := sc.Peek("k")
		assert.Equal(t, CacheStale, res.Status, "Peek must report a past-MaxAge entry as stale")
		assert.Equal(t, 1, sc.Sum(), "Peek must not evict entries")
	})
}

// BUG 5: Set on an existing key kept the original creation time, so a
// refreshed value inherited the old value's age. Since MaxAge and StaleAge
// both count from creation time, every refresh was still reported as stale
// (breaking the README's stale-while-revalidate pattern), and a regularly
// updated key would hard-expire a fixed time after it was first created.
//
// Timeline (StaleAge=50ms, MaxAge=100ms): set, sleep 80ms -> CacheStale,
// refresh, right after the refresh the entry must be CacheHit, and 80ms
// after that (160ms after the original Set, but only 80ms after the
// refresh) the entry must still be alive. The buggy code reports CacheStale
// immediately after the refresh and a Get miss at the end.
func TestSetUpdateResetsAge(t *testing.T) {
	staleAge := 50 * time.Millisecond
	maxAge := 100 * time.Millisecond
	sc := New[string](Option{StaleAge: &staleAge, MaxAge: &maxAge})

	_ = sc.Set("k", "old")
	time.Sleep(80 * time.Millisecond)

	res := sc.Peek("k")
	assert.Equal(t, CacheStale, res.Status, "setup: entry should be stale before the refresh")

	_ = sc.Set("k", "new") // e.g. the background refresh from the README

	res = sc.Peek("k")
	assert.Equal(t, CacheHit, res.Status, "a refreshed value must be fresh, not inherit the old value's age")
	assert.Equal(t, "new", res.Value)

	time.Sleep(80 * time.Millisecond)
	v, ok := sc.Get("k")
	assert.True(t, ok, "refreshed value expired as if it kept the original creation time")
	assert.Equal(t, "new", v)
	assert.Equal(t, 1, sc.Sum())
}

// evictLFU must sample exactly EvictionSamples entries. With EvictionSamples=1
// the victim is a single random entry, so the key that does NOT have the fewest
// hits (here "hot") is evicted about half the time. The buggy code used
// "count > evictionSamples" and therefore inspected two entries, always
// evicting "cold"; over 2000 trials that would produce 0 evictions of "hot",
// far below the threshold below.
func TestLFUEvictionSamplesExactlyConfiguredCount(t *testing.T) {
	ep := LFU
	mi := 2
	es := 1

	hotEvicted := 0
	for i := 0; i < 2000; i++ {
		sc := New[int](Option{MaxItems: &mi, EvictionPolicy: &ep, EvictionSamples: &es})
		_ = sc.Set("cold", 0)
		_ = sc.Set("hot", 1)
		_, _ = sc.Get("hot") // hit counts: cold=0, hot=1
		_ = sc.Set("new", 2) // cache full -> one eviction

		if _, ok := sc.Get("hot"); !ok {
			hotEvicted++
		}
	}

	// Binomial(2000, 0.5): P(hotEvicted < 200) is ~10^-183
	assert.Greater(t, hotEvicted, 200, "victim was never the non-minimum-hit key: more than EvictionSamples entries are being inspected")
}
