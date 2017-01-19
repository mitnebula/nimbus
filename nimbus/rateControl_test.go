package main

import (
	"testing"
	"time"
)

func BenchmarkRateUpdate(b *testing.B) {
	//setup - add a bunch of times to sendTimes, ackTimes, rtts
	rtt := time.Duration(150) * time.Millisecond
	now := time.Unix(946728000, 0)
	pkt := Packet{
		SeqNo:   0,
		VirtFid: 0,
	}
	sendTimes.UpdateDuration(rtt * 100)
	ackTimes.UpdateDuration(rtt * 100)
	for i := 0; i < 1000; i++ {
		sendTimes.Add(now, pkt)
		ackTimes.Add(now.Add(rtt), pkt)
		rtts.Add(durationLogVal(rtt))

		pkt.SeqNo++
		now = now.Add(rtt)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doUpdate()
	}
}
