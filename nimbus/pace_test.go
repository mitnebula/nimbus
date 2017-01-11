package main

import (
	"testing"
)

func BenchmarkPace(b *testing.B) {
	pacing := make(chan interface{})
	flowRate = 48e6
	go flowPacer(pacing)

	for i := 0; i < b.N; i++ {
		<-pacing
	}
}
