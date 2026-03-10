package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCacheMetricsHitRecording tests hit recording
func TestCacheMetricsHitRecording(t *testing.T) {
	cm := &CacheMetrics{}

	cm.RecordHit()
	cm.RecordHit()
	cm.RecordHit()

	hits, misses, inv := cm.GetStats()
	assert.Equal(t, int64(3), hits)
	assert.Equal(t, int64(0), misses)
	assert.Equal(t, int64(0), inv)
}

// TestCacheMetricsMissRecording tests miss recording
func TestCacheMetricsMissRecording(t *testing.T) {
	cm := &CacheMetrics{}

	cm.RecordHit()
	cm.RecordMiss()
	cm.RecordMiss()

	hits, misses, inv := cm.GetStats()
	assert.Equal(t, int64(1), hits)
	assert.Equal(t, int64(2), misses)
	assert.Equal(t, int64(0), inv)
}

// TestCacheMetricsInvalidationRecording tests invalidation recording
func TestCacheMetricsInvalidationRecording(t *testing.T) {
	cm := &CacheMetrics{}

	cm.RecordInvalidation()
	cm.RecordInvalidation()

	hits, misses, inv := cm.GetStats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(0), misses)
	assert.Equal(t, int64(2), inv)
}

// TestCacheMetricsHitRate tests hit rate calculation
func TestCacheMetricsHitRate(t *testing.T) {
	cm := &CacheMetrics{}

	// 2 hits, 3 misses = 40% hit rate
	cm.RecordHit()
	cm.RecordHit()
	cm.RecordMiss()
	cm.RecordMiss()
	cm.RecordMiss()

	hitRate := cm.HitRate()
	expected := 2.0 / 5.0
	assert.InDelta(t, expected, hitRate, 0.01)
}

// TestCacheMetricsZeroHitRate tests hit rate with no data
func TestCacheMetricsZeroHitRate(t *testing.T) {
	cm := &CacheMetrics{}
	assert.Equal(t, 0.0, cm.HitRate())
}

// TestCacheMetricsPerfectHitRate tests 100% hit rate
func TestCacheMetricsPerfectHitRate(t *testing.T) {
	cm := &CacheMetrics{}

	cm.RecordHit()
	cm.RecordHit()
	cm.RecordHit()

	assert.Equal(t, 1.0, cm.HitRate())
}

// TestCacheMetricsReset tests reset functionality
func TestCacheMetricsReset(t *testing.T) {
	cm := &CacheMetrics{}

	cm.RecordHit()
	cm.RecordHit()
	cm.RecordMiss()

	cm.Reset()

	hits, misses, inv := cm.GetStats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(0), misses)
	assert.Equal(t, int64(0), inv)
}

// TestOperationMetricsRecording tests operation recording
func TestOperationMetricsRecording(t *testing.T) {
	om := &OperationMetrics{}

	om.RecordOperation(100)
	om.RecordOperation(150)
	om.RecordOperation(200)

	count, total, min, max, avg := om.GetStats()
	assert.Equal(t, int64(3), count)
	assert.Equal(t, int64(450), total)
	assert.Equal(t, int64(100), min)
	assert.Equal(t, int64(200), max)
	assert.Equal(t, 150.0, avg)
}

// TestOperationMetricsSingleOperation tests single operation
func TestOperationMetricsSingleOperation(t *testing.T) {
	om := &OperationMetrics{}

	om.RecordOperation(50)

	count, total, min, max, avg := om.GetStats()
	assert.Equal(t, int64(1), count)
	assert.Equal(t, int64(50), total)
	assert.Equal(t, int64(50), min)
	assert.Equal(t, int64(50), max)
	assert.Equal(t, 50.0, avg)
}

// TestOperationMetricsReset tests reset
func TestOperationMetricsReset(t *testing.T) {
	om := &OperationMetrics{}

	om.RecordOperation(100)
	om.RecordOperation(200)

	om.Reset()

	count, total, min, max, avg := om.GetStats()
	assert.Equal(t, int64(0), count)
	assert.Equal(t, int64(0), total)
	assert.Equal(t, int64(0), min)
	assert.Equal(t, int64(0), max)
	assert.Equal(t, 0.0, avg)
}

// TestMeasurementTiming tests the Measurement helper
func TestMeasurementTiming(t *testing.T) {
	om := &OperationMetrics{}

	m := NewMeasurement(om)
	time.Sleep(10 * time.Millisecond)
	m.Record()

	count, _, _, _, _ := om.GetStats()
	assert.Equal(t, int64(1), count)
}

// TestConcurrentCacheMetrics tests thread-safety
func TestConcurrentCacheMetrics(t *testing.T) {
	cm := &CacheMetrics{}
	done := make(chan bool)

	// 5 goroutines, each recording 20 hits
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				cm.RecordHit()
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	hits, _, _ := cm.GetStats()
	assert.Equal(t, int64(100), hits)
}

// TestConcurrentOperationMetrics tests operation metrics thread-safety
func TestConcurrentOperationMetrics(t *testing.T) {
	om := &OperationMetrics{}
	done := make(chan bool)

	// 5 goroutines, each recording 10 operations
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				om.RecordOperation(int64(j + 1))
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	count, _, _, _, _ := om.GetStats()
	assert.Equal(t, int64(50), count)
}
