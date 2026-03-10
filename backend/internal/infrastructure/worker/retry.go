package worker

import "time"

// CalculateNextAttempt calculates when to retry based on exponential backoff
// Formula: delay = min(baseDelay * 2^attempts, maxDelay)
//
// Parameters:
//   - attempts: Number of previous attempts (0 for first retry)
//   - baseDelay: Initial retry delay (e.g., 1 second)
//   - maxDelay: Maximum retry delay cap (e.g., 60 seconds)
//
// Returns:
//   - time.Time: When the next retry attempt should occur
//
// Example progression with baseDelay=1s, maxDelay=60s:
//   - Attempt 0: 1s
//   - Attempt 1: 2s
//   - Attempt 2: 4s
//   - Attempt 3: 8s
//   - Attempt 4: 16s
//   - Attempt 5: 32s
//   - Attempt 6: 60s (capped)
//   - Attempt 7+: 60s (capped)
func CalculateNextAttempt(attempts int, baseDelay, maxDelay time.Duration) time.Time {
	// Calculate exponential backoff delay
	delay := baseDelay
	for i := 0; i < attempts; i++ {
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
			break
		}
	}

	return time.Now().Add(delay)
}
