package main

import (
	"math"
	"testing"
	"time"
)

func TestPace(t *testing.T) {
	pacing := make(chan interface{})
	flowRate = 12e6
	go flowPacer(pacing)

	start := time.Now()
	for i := 0; i < 10000; i++ {
		<-pacing
	}

	dur := time.Duration(time.Since(start).Nanoseconds()/10000) * time.Nanosecond
	drift := time.Duration(1)*time.Millisecond - dur

	if math.Abs(float64(drift.Nanoseconds())) > 500.0 {
		t.Error("pacing drifted more than 0.5us", drift)
	}
}
