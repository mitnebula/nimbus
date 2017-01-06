package main

import (
	"fmt"
	"math"
	"time"
)

func MakeBytes(size int) string {
	result := make([]byte, 0, size)
	for i := 0; i < size; i++ {
		result = append(result, 'a')
	}

	return string(result)
}

func Now() int64 {
	return time.Now().UnixNano()
}

func ThroughputFromTimes(times *TimedLog, now time.Time, delay time.Duration) (float64, error) {
	times.mux.Lock()
	defer times.mux.Unlock()

	newest, _, err := times.Before(now)
	if err != nil {
		return 0, err
	}

	oldest, ot, err := times.Before(now.Add(-1 * delay))
	if err != nil {
		return 0, err
	}

	for oldest == newest {
		// get the next oldest time
		oldest, _, err = times.Before(ot.Add(-1 * time.Nanosecond))
		if err != nil {
			return 0, err
		}
	}

	numSent, err := times.CountBetween(now, now.Add(-1*delay))
	if err != nil {
		return 0, err
	}

	tot := float64(numSent * 1480 * 8.0)
	//dur := time.Unix(0, int64(newest.(intLogVal))).Sub(time.Unix(0, int64(oldest.(intLogVal))))
	dur := float64(int64(newest.(intLogVal))-int64(oldest.(intLogVal))) / 1e9
	tpt := tot / dur
	//fmt.Println(numSent, times.Len(), newest, oldest, dur, tpt)
	if math.IsNaN(tpt) || math.IsInf(tpt, 1) || tpt < 0 {
		return 0, fmt.Errorf("undefined throughput: %v %v", tot, dur)
	}

	return tpt, nil
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
