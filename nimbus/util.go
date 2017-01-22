package main

import (
	"fmt"
	"math"
	"time"
)

func Now() int64 {
	return time.Now().UnixNano()
}

func ThroughputFromTimes(
	times *PacketLog,
	now time.Time,
	delay time.Duration,
) (float64, Packet, Packet, error) {
	times.mux.Lock()
	defer times.mux.Unlock()

	if times.Len() < 2 {
		return 0, Packet{}, Packet{}, fmt.Errorf("not enough values")
	}

	newestPkt, newestPktTime, err := times.Before(now)
	if err != nil {
		return 0, Packet{}, Packet{}, err
	}

	oldestPkt, oldestPktTime, err := times.Before(now.Add(-1 * delay))
	if err != nil {
		return 0, Packet{}, Packet{}, err
	}

	for newestPktTime.Equal(oldestPktTime) {
		oldestPkt, oldestPktTime, err = times.Before(newestPktTime.Add(-1 * time.Nanosecond))
		if err != nil {
			return 0, Packet{}, Packet{}, err
		}
	}

	dur := newestPktTime.Sub(oldestPktTime).Seconds()
	cnt, _ := times.NumItemsBetween(oldestPktTime, newestPktTime)
	tot := float64(cnt * ONE_PACKET)
	tpt := tot / dur
	if math.IsNaN(tpt) || math.IsInf(tpt, 1) || tpt < 0 {
		return 0, Packet{}, Packet{}, fmt.Errorf("undefined throughput: %v %v", tot, dur)
	}

	return tpt, oldestPkt, newestPkt, nil
}

func PacketTimes(
	times *PacketLog,
	oldPkt Packet,
	newPkt Packet,
) (time.Time, time.Time, error) {
	// set to 0 to make it match in the map
	oldPkt.RecvTime = 0
	newPkt.RecvTime = 0
	oldPkt.Echo = 0
	newPkt.Echo = 0

	times.mux.Lock()
	defer times.mux.Unlock()

	oldPktTime, ok := times.m[oldPkt]
	if !ok {
		t := time.Now()
		return t, t, fmt.Errorf("can't find packet time: %v", oldPkt)
	}
	newPktTime, ok := times.m[newPkt]
	if !ok {
		t := time.Now()
		return t, t, fmt.Errorf("can't find packet time: %v", newPkt)
	}

	return newPktTime, oldPktTime, nil
}

func MinRtt(rtts *Log) time.Duration {
	var min_rtt time.Duration
	lv, err := rtts.Min()
	if err != nil {
		min_rtt, _ = time.ParseDuration("0s")
	}
	min_rtt = time.Duration(lv.(durationLogVal))
	return min_rtt
}

func PrintPacket(pkt Packet) string {
	return fmt.Sprintf(
		"{seq %d vfid %d echo %d recv %d size %d}",
		pkt.SeqNo,
		pkt.VirtFid,
		pkt.Echo,
		pkt.RecvTime,
		len(pkt.Payload),
	)
}
