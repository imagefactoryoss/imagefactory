package metrics

import (
	"sync"
	"sync/atomic"
)

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	Hits         int64
	Misses       int64
	Invalidations int64
	mu           sync.RWMutex
}

// RecordHit increments hit counter
func (cm *CacheMetrics) RecordHit() {
	atomic.AddInt64(&cm.Hits, 1)
}

// RecordMiss increments miss counter
func (cm *CacheMetrics) RecordMiss() {
	atomic.AddInt64(&cm.Misses, 1)
}

// RecordInvalidation increments invalidation counter
func (cm *CacheMetrics) RecordInvalidation() {
	atomic.AddInt64(&cm.Invalidations, 1)
}

// GetStats returns a snapshot of metrics
func (cm *CacheMetrics) GetStats() (hits, misses, invalidations int64) {
	hits = atomic.LoadInt64(&cm.Hits)
	misses = atomic.LoadInt64(&cm.Misses)
	invalidations = atomic.LoadInt64(&cm.Invalidations)
	return
}

// HitRate returns the cache hit rate (0.0 to 1.0)
func (cm *CacheMetrics) HitRate() float64 {
	hits := atomic.LoadInt64(&cm.Hits)
	misses := atomic.LoadInt64(&cm.Misses)
	
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total)
}

// Reset clears all metrics
func (cm *CacheMetrics) Reset() {
	atomic.StoreInt64(&cm.Hits, 0)
	atomic.StoreInt64(&cm.Misses, 0)
	atomic.StoreInt64(&cm.Invalidations, 0)
}
