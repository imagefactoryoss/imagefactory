package worker

import (
	"testing"
	"time"
)

func TestCalculateNextAttempt(t *testing.T) {
	base := time.Second
	max := 10 * time.Second
	start := time.Now()

	next0 := CalculateNextAttempt(0, base, max)
	if next0.Before(start.Add(base)) || next0.After(start.Add(2*base)) {
		t.Fatalf("attempt0 delay out of expected range: %v", next0.Sub(start))
	}

	next2 := CalculateNextAttempt(2, base, max) // 4s
	if next2.Before(start.Add(4*time.Second)) || next2.After(start.Add(5*time.Second)) {
		t.Fatalf("attempt2 delay out of expected range: %v", next2.Sub(start))
	}

	next10 := CalculateNextAttempt(10, base, max) // capped at max
	if next10.Before(start.Add(max)) || next10.After(start.Add(max+time.Second)) {
		t.Fatalf("attempt10 delay should be capped near max, got %v", next10.Sub(start))
	}
}
