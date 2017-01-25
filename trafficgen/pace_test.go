package main

import (
	"testing"
	"time"
)

func BenchmarkPace(b *testing.B) {
	rtts.Add(durationLogVal(time.Duration(20) * time.Millisecond))
	rtts.Add(durationLogVal(time.Duration(20) * time.Millisecond))
	rtts.Add(durationLogVal(time.Duration(20) * time.Millisecond))

	b.ResetTimer()
	pacing := make(chan interface{})
	flowRate = 48e6
	go flowPacer(pacing)

	for i := 0; i < b.N; i++ {
		<-pacing
	}
}
