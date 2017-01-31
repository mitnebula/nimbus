package main

import (
	"fmt"
	"math"
	"time"

	"github.com/akshayknarayan/history"
	"github.com/montanaflynn/stats"
)

func Now() int64 {
	return time.Now().UnixNano()
}

func NowPretty() string {
	return time.Now().Format("15:04:05.000000")
}

func BpsToMbps(bps float64) float64 {
	return (bps / 1000000)
}

func VarianceFromTimes(
	times *history.UniqueHistory,
	now time.Time,
	delay time.Duration,
) (float64, error) {
	if times.Len() < 2 {
		return 0, fmt.Errorf("not enough values")
	}

	its, err := times.ItemsBetween(now.Add(-1*delay), now)
	if err != nil {
		return 0, err
	}

	var inters []float64
	curr := its[0].Time
	for _, i := range its {
		inters = append(inters, i.Time.Sub(curr).Seconds())
		curr = i.Time
	}

	inters = inters[1:]
	return stats.Variance(inters)
}

func ThroughputFromTimes(
	times *history.UniqueHistory,
	now time.Time,
	delay time.Duration,
) (float64, Packet, Packet, error) {
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

	return tpt, oldestPkt.(Packet), newestPkt.(Packet), nil
}

func PacketTimes(
	times *history.UniqueHistory,
	oldPkt Packet,
	newPkt Packet,
) (time.Time, time.Time, error) {
	// set to 0 to make it match in the map
	oldPkt.RecvTime = 0
	newPkt.RecvTime = 0
	oldPkt.Echo = 0
	newPkt.Echo = 0

	oldPktTime, err := times.TimeOf(oldPkt)
	if err != nil {
		t := time.Now()
		return t, t, fmt.Errorf("can't find packet time: %v: %v", oldPkt, err)
	}
	newPktTime, err := times.TimeOf(newPkt)
	if err != nil {
		t := time.Now()
		return t, t, fmt.Errorf("can't find packet time: %v: %v", oldPkt, err)
	}

	return newPktTime, oldPktTime, nil
}

func PrintPacket(pkt Packet) string {
	return fmt.Sprintf(
		"{echo %d recv %d size %d}",
		pkt.Echo,
		pkt.RecvTime,
		len(pkt.Payload),
	)
}
