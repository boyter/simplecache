// SPDX-License-Identifier: MIT

package simplecache

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSimpleCache_Interface(t *testing.T) {
	type CacheTest struct {
		Cache CacheInterface[string]
	}

	_ = CacheTest{Cache: &Cache[string]{}}
}

func TestSimpleCache_GetNothing(t *testing.T) {
	type Sample struct {
		Key string
	}

	type testCase[T any] struct {
		name          string
		expectedCount int
	}
	tests := []testCase[Sample]{
		{
			name:          "expect nothing",
			expectedCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := New[Sample]()

			for i := 0; i < 1000; i++ {
				_, ok := sc.Get(fmt.Sprintf("key-%d", i))
				assert.False(t, ok)
			}

			assert.Equal(t, tt.expectedCount, sc.Sum())
		})
	}
}

func TestSimpleCache_MaxItemsOption(t *testing.T) {
	type Sample struct {
		Key string
	}

	type testCase[T any] struct {
		name          string
		expectedCount int
	}
	tests := []testCase[Sample]{
		{
			name:          "max items one",
			expectedCount: 1,
		},
		{
			name:          "max items two",
			expectedCount: 2,
		},
		{
			name:          "max items 100",
			expectedCount: 100,
		},
		{
			name:          "max items 50",
			expectedCount: 50,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := New[Sample](Option{
				MaxItems: &tt.expectedCount,
			})

			for i := 0; i < tt.expectedCount+2; i++ {
				_ = sc.Set(fmt.Sprintf("key-%d", i), Sample{})
			}

			assert.Equal(t, tt.expectedCount, sc.Sum())
		})
	}
}

func TestSimpleCache_Option(t *testing.T) {
	type Sample struct {
		Key string
	}

	type testCase[T any] struct {
		name string
	}
	tests := []testCase[Sample]{
		{
			name: "empty all options",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oEp := LRU
			oEs := 5
			oMA := time.Nanosecond

			_ = New[Sample](Option{
				MaxItems:        &oEs,
				EvictionPolicy:  &oEp,
				EvictionSamples: &oEs,
				MaxAge:          &oMA,
			})
		})
	}
}

func TestSimpleCache_DefaultSet(t *testing.T) {
	type Sample struct {
		Key string
	}

	type args struct {
		values []Sample
	}
	type testCase[T any] struct {
		name          string
		args          args
		expectedCount int
	}
	tests := []testCase[Sample]{
		{
			name: "add one get one",
			args: args{
				values: []Sample{{
					Key: "test",
				}},
			},
			expectedCount: 1,
		},
		{
			name: "add one get another",
			args: args{
				values: []Sample{{
					Key: "test2",
				}},
			},
			expectedCount: 1,
		},
		{
			name: "add two check get",
			args: args{
				values: []Sample{{
					Key: "test1",
				}, {
					Key: "test2",
				}},
			},
			expectedCount: 2,
		},
		{
			name: "add three check get",
			args: args{
				values: []Sample{{
					Key: "test1",
				}, {
					Key: "test2",
				}, {
					Key: "test3",
				}},
			},
			expectedCount: 3,
		},
		{
			name: "add three duplicate get 2",
			args: args{
				values: []Sample{{
					Key: "test1",
				}, {
					Key: "test2",
				}, {
					Key: "test2",
				}},
			},
			expectedCount: 2,
		},
		{
			name: "add three duplicate get 1",
			args: args{
				values: []Sample{{
					Key: "test1",
				}, {
					Key: "test1",
				}, {
					Key: "test1",
				}},
			},
			expectedCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := New[Sample]()
			for _, s := range tt.args.values {
				_ = sc.Set(s.Key, s)
			}
			for _, s := range tt.args.values {
				v, ok := sc.Get(s.Key)
				assert.True(t, ok)
				assert.Equal(t, s.Key, v.Key)
			}

			assert.Equal(t, tt.expectedCount, sc.Sum())
		})
	}
}

func TestSimpleCache_Expire(t *testing.T) {
	type Sample struct {
		Key string
	}

	type args struct {
		values []Sample
	}
	type testCase[T any] struct {
		name          string
		args          args
		expectedCount int
	}
	tests := []testCase[Sample]{
		{
			name: "add one get nothing",
			args: args{
				values: []Sample{{
					Key: "test",
				}},
			},
			expectedCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxAge := time.Nanosecond
			sc := New[Sample](Option{
				MaxAge: &maxAge,
			})
			for _, s := range tt.args.values {
				_ = sc.Set(s.Key, s)
			}

			for _, s := range tt.args.values {
				_, ok := sc.Get(s.Key)
				assert.False(t, ok)
			}

			assert.Equal(t, tt.expectedCount, sc.Sum())
		})
	}
}

func TestSimpleCache_Delete(t *testing.T) {
	type Sample struct {
		Key string
	}

	type args struct {
		values []Sample
	}
	type testCase[T any] struct {
		name          string
		args          args
		expectedCount int
	}
	tests := []testCase[Sample]{
		{
			name: "add one delete one",
			args: args{
				values: []Sample{{
					Key: "test",
				}},
			},
			expectedCount: 0,
		},
		{
			name: "add two delete two",
			args: args{
				values: []Sample{{
					Key: "test",
				}, {
					Key: "test2",
				}},
			},
			expectedCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxAge := time.Nanosecond
			sc := New[Sample](Option{
				MaxAge: &maxAge,
			})
			for _, s := range tt.args.values {
				_ = sc.Set(s.Key, s)
			}

			for _, s := range tt.args.values {
				sc.Delete(s.Key)
			}

			assert.Equal(t, tt.expectedCount, sc.Sum())
		})
	}
}

func TestSimpleCache_Clear(t *testing.T) {
	type Sample struct {
		Key string
	}

	type testCase[T any] struct {
		name          string
		expectedCount int
	}
	tests := []testCase[Sample]{
		{
			name:          "add one get nothing",
			expectedCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := New[Sample]()

			for i := 0; i < 100; i++ {
				_ = sc.Set(fmt.Sprintf("key-%d", i), Sample{})
			}

			sc.Clear()
			assert.Equal(t, tt.expectedCount, sc.Sum())
		})
	}
}

func TestSimpleCache_LRU(t *testing.T) {
	ep := LRU
	mi := 5
	sc := New[int](Option{
		MaxItems:       &mi,
		EvictionPolicy: &ep,
	})

	for i := 0; i < 5; i++ {
		_ = sc.Set(fmt.Sprintf("key-%d", i), i)
	}

	for j := 0; j < 50; j++ {
		for i := 0; i < 4; i++ {
			_ = sc.Set(fmt.Sprintf("key-%d", i), i)
		}
	}

	_ = sc.Set(fmt.Sprintf("key-%d", 6), 6)

	_, ok := sc.Get(fmt.Sprintf("key-%d", 5))

	assert.False(t, ok)
}

func TestSimpleCache_GetWithStatus_Hit(t *testing.T) {
	staleAge := 10 * time.Second
	maxAge := 20 * time.Second
	sc := New[string](Option{
		StaleAge: &staleAge,
		MaxAge:   &maxAge,
	})

	_ = sc.Set("key1", "value1")
	result := sc.GetWithStatus("key1")
	assert.Equal(t, CacheHit, result.Status)
	assert.Equal(t, "value1", result.Value)
}

func TestSimpleCache_GetWithStatus_Stale(t *testing.T) {
	staleAge := time.Nanosecond
	maxAge := time.Hour
	sc := New[string](Option{
		StaleAge: &staleAge,
		MaxAge:   &maxAge,
	})

	_ = sc.Set("key1", "value1")
	time.Sleep(time.Millisecond)
	result := sc.GetWithStatus("key1")
	assert.Equal(t, CacheStale, result.Status)
	assert.Equal(t, "value1", result.Value)
}

func TestSimpleCache_GetWithStatus_Miss_Expired(t *testing.T) {
	staleAge := time.Nanosecond
	maxAge := time.Nanosecond
	sc := New[string](Option{
		StaleAge: &staleAge,
		MaxAge:   &maxAge,
	})

	_ = sc.Set("key1", "value1")
	time.Sleep(time.Millisecond)
	result := sc.GetWithStatus("key1")
	assert.Equal(t, CacheMiss, result.Status)
	assert.Equal(t, "", result.Value)
	assert.Equal(t, 0, sc.Sum())
}

func TestSimpleCache_GetWithStatus_Miss_NotFound(t *testing.T) {
	sc := New[string]()
	result := sc.GetWithStatus("nonexistent")
	assert.Equal(t, CacheMiss, result.Status)
	assert.Equal(t, "", result.Value)
}

func TestSimpleCache_GetWithStatus_NoStaleAge(t *testing.T) {
	sc := New[string]()

	_ = sc.Set("key1", "value1")
	result := sc.GetWithStatus("key1")
	assert.Equal(t, CacheHit, result.Status)
	assert.Equal(t, "value1", result.Value)
}

func TestSimpleCache_GetWithStatus_GetUnchanged(t *testing.T) {
	staleAge := time.Nanosecond
	maxAge := time.Hour
	sc := New[string](Option{
		StaleAge: &staleAge,
		MaxAge:   &maxAge,
	})

	_ = sc.Set("key1", "value1")
	time.Sleep(time.Millisecond)

	// Get should still work as before (returns value, true) regardless of staleAge
	val, ok := sc.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestSimpleCache_Peek_Hit(t *testing.T) {
	staleAge := 10 * time.Second
	maxAge := 20 * time.Second
	sc := New[string](Option{
		StaleAge: &staleAge,
		MaxAge:   &maxAge,
	})

	_ = sc.Set("key1", "value1")
	result := sc.Peek("key1")
	assert.Equal(t, CacheHit, result.Status)
	assert.Equal(t, "value1", result.Value)
}

func TestSimpleCache_Peek_Stale(t *testing.T) {
	staleAge := time.Nanosecond
	maxAge := time.Hour
	sc := New[string](Option{
		StaleAge: &staleAge,
		MaxAge:   &maxAge,
	})

	_ = sc.Set("key1", "value1")
	time.Sleep(time.Millisecond)
	result := sc.Peek("key1")
	assert.Equal(t, CacheStale, result.Status)
	assert.Equal(t, "value1", result.Value)
}

func TestSimpleCache_Peek_Expired_NotDeleted(t *testing.T) {
	staleAge := time.Nanosecond
	maxAge := time.Nanosecond
	sc := New[string](Option{
		StaleAge: &staleAge,
		MaxAge:   &maxAge,
	})

	_ = sc.Set("key1", "value1")
	time.Sleep(time.Millisecond)
	result := sc.Peek("key1")
	// Peek returns CacheStale (not CacheMiss) and does NOT delete
	assert.Equal(t, CacheStale, result.Status)
	assert.Equal(t, "value1", result.Value)
	assert.Equal(t, 1, sc.Sum(), "Peek must not evict entries")
}

func TestSimpleCache_Peek_Miss_NotFound(t *testing.T) {
	sc := New[string]()
	result := sc.Peek("nonexistent")
	assert.Equal(t, CacheMiss, result.Status)
	assert.Equal(t, "", result.Value)
}

func TestSimpleCache_NoExpiration(t *testing.T) {
	minusExpire := -time.Hour
	ep := LRU
	mi := 5
	sc := New[int](Option{
		MaxItems:       &mi,
		EvictionPolicy: &ep,
		MaxAge:         &minusExpire,
	})
	for i := 0; i < 5; i++ {
		_ = sc.Set(fmt.Sprintf("key-%d", i), i)
	}
	for i := 0; i < 5; i++ {
		v, ok := sc.Get(fmt.Sprintf("key-%d", i))
		assert.True(t, ok)
		assert.Equal(t, v, i)
	}

	zeroExpire := time.Duration(0)
	sc = New[int](Option{
		MaxItems:       &mi,
		EvictionPolicy: &ep,
		MaxAge:         &zeroExpire,
	})
	for i := 0; i < 5; i++ {
		_ = sc.Set(fmt.Sprintf("key-%d", i), i)
	}
	for i := 0; i < 5; i++ {
		v, ok := sc.Get(fmt.Sprintf("key-%d", i))
		assert.True(t, ok)
		assert.Equal(t, v, i)
	}
}
